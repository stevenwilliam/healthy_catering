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
	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/config"
	"github.com/stevenwilliam/healthy_catering/internal/platform/database"
	"github.com/stevenwilliam/healthy_catering/internal/platform/logging"
	"github.com/stevenwilliam/healthy_catering/internal/platform/migrate"
	"github.com/stevenwilliam/healthy_catering/internal/platform/ratelimit"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

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
	default:
		return fmt.Errorf("unknown command %q (serve|migrate|create-staff|version)", cmd)
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

	signer := security.NewTokenSigner(
		cfg.Auth.SigningKey, cfg.Auth.PreviousKey, cfg.Auth.Issuer,
		15*time.Minute, time.Now)

	auth := app.NewAuth(app.AuthDeps{
		Users: users, Audit: audit, Params: params, Signer: signer,
	})

	limiter := ratelimit.New(time.Now)

	router := adapterhttp.New(adapterhttp.Deps{
		Config:         cfg,
		Log:            log,
		Serviceability: app.NewServiceability(kitchens, params, tz),
		Auth:           auth,
		Admin:          app.NewAdmin(master, settings, audit, params),
		Signer:         signer,
		Limiter:        limiter,
		// Until the mailer exists (M11), the verification link is logged at
		// debug rather than silently dropped — so a developer can complete the
		// flow, and so it is obvious this is not yet a real email.
		OnVerificationToken: func(userID uuid.UUID, token string) {
			log.Warn("email verification token issued but NOT emailed — mailer lands in M11",
				"user_id", userID, "verify_url",
				fmt.Sprintf("%s/verify-email?token=%s", cfg.App.BaseURL, token))
		},
		Health: func() error {
			c, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			return sqlDB.PingContext(c)
		},
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening",
			"port", cfg.App.Port, "env", cfg.App.Env, "version", version,
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
