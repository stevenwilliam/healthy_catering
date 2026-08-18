// Command api is the Evermore service: a thin entrypoint that wires the
// dependencies and runs one of the subcommands (CLAUDE.md §2).
//
//	api serve     run the HTTP server
//	api migrate   apply pending migrations (up|down|status)
//	api create-staff --email <a> --role <r>   first-run setup; no default admin
//	api version   print the build version
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	adapterhttp "github.com/stevenwilliam/healthy_catering/internal/adapter/http"
	"github.com/stevenwilliam/healthy_catering/internal/adapter/notify"
	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/adapter/storage"
	"github.com/stevenwilliam/healthy_catering/internal/adapter/translate"
	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/config"
	"github.com/stevenwilliam/healthy_catering/internal/platform/database"
	"github.com/stevenwilliam/healthy_catering/internal/platform/logging"
	"github.com/stevenwilliam/healthy_catering/internal/platform/migrate"
	"github.com/stevenwilliam/healthy_catering/internal/platform/ratelimit"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// firstNonEmpty prefers the environment over the database, so a production
// secret can stay out of the database entirely while development reads the
// back-office setting.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// version is set at build time: -ldflags "-X main.version=$(git rev-parse --short HEAD)".
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "evermore: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd == "version" {
		fmt.Println(version)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(cfg.App.LogLevel, cfg.App.IsProduction())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	gdb, err := database.Open(ctx, database.Options{
		URL:          cfg.Database.URL,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		Debug:        !cfg.App.IsProduction(),
	}, log)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer sqlDB.Close()

	switch cmd {
	case "migrate":
		return runMigrate(ctx, sqlDB, log)
	case "serve":
		return serve(ctx, cfg, gdb, log)
	case "create-staff":
		return runCreateStaff(ctx, gdb, log)
	case "seed-menu":
		days, err := seedDays(os.Args)
		if err != nil {
			return err
		}
		return runSeedMenu(ctx, gdb, log, days)
	default:
		return fmt.Errorf("unknown command %q (serve|migrate|create-staff|seed-menu|version)", cmd)
	}
}

func runMigrate(ctx context.Context, sqlDB *sql.DB, log *slog.Logger) error {
	direction := "up"
	if len(os.Args) > 2 {
		direction = os.Args[2]
	}
	switch direction {
	case "up":
		n, err := migrate.Up(ctx, sqlDB, log)
		if err != nil {
			return err
		}
		log.Info("migrations applied", "count", n)
		return nil
	case "down":
		// Forward-only in production (CLAUDE.md §4); down exists for
		// development, CI and a rollback of last resort.
		if os.Getenv("APP_ENV") == "production" && os.Getenv("ALLOW_MIGRATE_DOWN") != "yes" {
			return errors.New("migrations are forward-only in production; set ALLOW_MIGRATE_DOWN=yes to override")
		}
		steps := 1
		if len(os.Args) > 3 {
			n, err := strconv.Atoi(os.Args[3])
			if err != nil || n < 1 {
				return fmt.Errorf("migrate down: step count must be a positive integer, got %q", os.Args[3])
			}
			steps = n
		}
		n, err := migrate.Down(ctx, sqlDB, steps, log)
		if err != nil {
			return err
		}
		log.Info("migrations rolled back", "count", n)
		return nil
	case "status":
		st, err := migrate.Current(ctx, sqlDB)
		if err != nil {
			return err
		}
		fmt.Printf("applied: %v\npending: %v\n", st.Applied, st.Pending)
		return nil
	default:
		return fmt.Errorf("unknown migrate direction %q (up|down|status)", direction)
	}
}

func serve(ctx context.Context, cfg *config.Config, gdb *gorm.DB, log *slog.Logger) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}

	tz, err := cfg.Locale.TZ()
	if err != nil {
		return fmt.Errorf("timezone: %w", err)
	}

	// Parameters are cached for 30s and invalidated on write: a stale cut-off
	// for a few seconds is harmless, for an hour it is a support call.
	params := sysparam.NewStore(sqlDB, 30*time.Second)

	kitchens := postgres.NewKitchenRepo(gdb)
	users := postgres.NewUserRepo(gdb)
	audit := postgres.NewAuditRepo(gdb)
	master := postgres.NewMasterDataRepo(gdb)
	settings := postgres.NewSettingsRepo(gdb)
	catalogue := postgres.NewCatalogueRepo(gdb)
	sched := postgres.NewScheduleRepo(gdb)
	pricingRepo := postgres.NewPricingRepo(gdb)
	orders := postgres.NewOrderRepo(gdb)
	payments := postgres.NewPaymentRepo(gdb)
	creditsRepo := postgres.NewCreditRepo(gdb)
	reportsRepo := postgres.NewReportRepo(gdb)
	deliveriesRepo := postgres.NewDeliveryRepo(gdb)

	signer := security.NewTokenSigner(
		cfg.Auth.SigningKey, cfg.Auth.PreviousKey, cfg.Auth.Issuer,
		15*time.Minute, time.Now)

	auth := app.NewAuth(app.AuthDeps{
		Users: users, Audit: audit, Params: params, Signer: signer,
	})

	// Staff second factor. Without TOTP_ENCRYPTION_KEY the secrets could only
	// be stored in the clear, so the feature stays OFF rather than storing them
	// badly — the routes are simply absent and the log says why.
	var mfa *app.MFA
	if totpCipher, err := security.NewTOTPCipher(cfg.Auth.TOTPKey); err != nil {
		log.Warn("two-factor authentication is disabled",
			"reason", "TOTP_ENCRYPTION_KEY is not set")
	} else {
		mfa = app.NewMFA(postgres.NewTOTPRepo(gdb), users, audit, totpCipher,
			cfg.Auth.Issuer, time.Now)
		auth.AttachMFA(mfa)
	}

	// WhatsApp, wired like mail: gateway settings live in sys_parameters so they
	// change without a deploy, and the env wins for the API KEY so the secret
	// never has to sit in a migration (CLAUDE.md §4).
	//
	// NewWAHA returns nil when it is not configured, and Multi simply has no
	// WhatsApp sender then — messages are never queued rather than queued and
	// permanently failing.
	whatsapp := notify.NewWAHA(notify.WAHAConfig{
		BaseURL: firstNonEmpty(cfg.WhatsApp.WAHAURL,
			params.String(ctx, sysparam.KeyWAHAURL, "http://127.0.0.1:3000")),
		Session: firstNonEmpty(cfg.WhatsApp.WAHASession,
			params.String(ctx, sysparam.KeyWAHASession, "default")),
		APIKey: firstNonEmpty(cfg.WhatsApp.WAHAAPIKey,
			params.String(ctx, sysparam.KeyWAHAAPIKey, "")),
	})
	if whatsapp == nil {
		log.Warn("whatsapp is disabled", "reason", "no WAHA gateway or API key configured")
	}

	limiter := ratelimit.New(time.Now)

	// Notifications: mail settings come from sys_parameters so Steven can change
	// the relay without a deploy (D14/B7); the env overrides for production.
	mailCfg := notify.SMTPConfig{
		Host:      firstNonEmpty(cfg.Mail.Host, params.String(ctx, sysparam.KeyMailHost, "127.0.0.1")),
		Port:      cfg.Mail.Port,
		Username:  firstNonEmpty(cfg.Mail.Username, params.String(ctx, sysparam.KeyMailUsername, "")),
		Password:  firstNonEmpty(cfg.Mail.Password, params.String(ctx, sysparam.KeyMailPassword, "")),
		FromEmail: firstNonEmpty(cfg.Mail.FromEmail, params.String(ctx, sysparam.KeyMailFromEmail, "no-reply@evermore.co.id")),
		FromName:  firstNonEmpty(cfg.Mail.FromName, params.String(ctx, sysparam.KeyMailFromName, "Evermore")),
		UseTLS:    cfg.Mail.TLS,
	}
	// Object storage. Payment proofs are photographs of bank accounts, so the
	// bucket is private and served only by presigned URL (99 §7).
	var objectStore *storage.Store
	if cfg.Storage.AccessKey != "" {
		objectStore, err = storage.New(ctx, storage.Config{
			Endpoint: cfg.Storage.Endpoint, PublicEndpoint: cfg.Storage.PublicEndpoint,
			AccessKey: cfg.Storage.AccessKey, SecretKey: cfg.Storage.SecretKey,
			Bucket: cfg.Storage.Bucket, UseSSL: cfg.Storage.UseSSL,
		})
		if err != nil {
			// In production this is fatal: an order that cannot take a proof
			// cannot be paid.
			if cfg.App.IsProduction() {
				return fmt.Errorf("object storage: %w", err)
			}
			log.Warn("object storage unavailable; upload routes are disabled", "error", err)
		} else {
			log.Info("object storage ready", "bucket", cfg.Storage.Bucket)
		}
	} else {
		log.Warn("no object storage credentials; upload routes are disabled")
	}

	jobsRepo := postgres.NewJobRepo(gdb)
	notifier := app.NewNotifier(app.NotifierDeps{
		Jobs:    jobsRepo,
		Senders: notify.NewMulti(notify.NewSMTPSender(mailCfg), whatsapp),
		Params:  params, Log: log, TZ: tz, BaseURL: cfg.App.BaseURL,
	})

	// One worker in-process. A second node would need a shared lock; the
	// FOR UPDATE SKIP LOCKED claim already makes that safe when it happens.
	go notifier.Run(ctx, 5*time.Second)

	// The expiry sweep is the only automated cancellation in the system, and it
	// touches unpaid orders only (99 §8).
	financeSvc := app.NewFinance(payments, creditsRepo, audit, tz, time.Now)
	financeSvc.OnPaid = func(c context.Context, orderID, customerID uuid.UUID, orderType string) {
		email, name, locale := users.ContactForCustomer(c, customerID)
		code, _ := orders.OrderCodeOf(c, orderID)
		extra := "Kami akan mengabari Anda menjelang pengiriman."
		if orderType == "PACKAGE" {
			extra = "Kredit Anda sudah aktif — silakan pilih jadwal makan Anda."
		}
		notifier.PaymentVerified(c, email, name, locale, orderID, code, extra)
	}
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := financeSvc.ExpireUnpaid(ctx); err != nil {
					log.Error("expiry sweep", "error", err)
				} else if n > 0 {
					log.Info("unpaid orders expired, capacity released", "count", n)
				}
			}
		}
	}()

	serviceability := app.NewServiceability(kitchens, params, tz)
	pricingSvc := app.NewPricing(pricingRepo, audit, params, tz)
	ordering := app.NewOrdering(app.OrderingDeps{
		Orders: orders, Payments: payments, Deliveries: deliveriesRepo, Notifier: notifier,
		Schedule: sched, Kitchens: kitchens, Users: users,
		Pricing: pricingSvc, Service: serviceability, Audit: audit,
		Params: params, TZ: tz,
	})

	// Machine translation of public copy. Off unless a provider is configured;
	// the back office still works without it, translations just wait for a
	// human (docs/11 §6).
	translator := translate.New(cfg.Translate.Provider, cfg.Translate.APIKey)
	if translator.Available() {
		log.Info("translation enabled", "provider", translator.Name())
	} else {
		log.Info("translation disabled",
			"reason", "no TRANSLATE_PROVIDER/TRANSLATE_API_KEY configured",
			"effect", "public copy must be translated by hand in the back office")
	}
	contentSvc := app.NewContent(app.ContentDeps{
		Repo: postgres.NewContentRepo(gdb), Translator: translator, Audit: audit, Log: log,
	})

	router := adapterhttp.New(adapterhttp.Deps{
		Config:         cfg,
		Log:            log,
		Serviceability: serviceability,
		Auth:           auth,
		Admin:          app.NewAdmin(master, settings, audit, params),
		Catalogue:      app.NewCatalogue(catalogue, sched, master, audit, params, tz),
		Pricing:        pricingSvc,
		Ordering:       ordering,
		Finance:        financeSvc,
		Notifier:       notifier,
		Storage:        objectStore,
		Fulfilment:     app.NewFulfilment(deliveriesRepo, creditsRepo, audit, params, tz, time.Now),
		Content:        contentSvc,
		MFA:            mfa,
		Reports:        app.NewReports(reportsRepo, params, tz),
		Params:         params,
		Packages: app.NewPackages(app.PackagesDeps{
			Credits: creditsRepo, Orders: orders, Schedule: sched, Users: users,
			Pricing: pricingSvc, Service: serviceability, Audit: audit,
			Params: params, TZ: tz,
		}),
		Signer:  signer,
		Limiter: limiter,
		OnVerificationToken: func(userID uuid.UUID, token string) {
			email, name, locale := users.ContactFor(ctx, userID)
			notifier.VerifyEmail(ctx, email, name, token, locale)
		},
		Health: func() error {
			c, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			return sqlDB.PingContext(c)
		},
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.App.Bind, cfg.App.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening",
			"bind", cfg.App.Bind, "port", cfg.App.Port, "env", cfg.App.Env, "version", version,
			"timezone", cfg.Locale.Timezone)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}
