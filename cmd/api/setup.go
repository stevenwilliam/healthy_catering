package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"golang.org/x/term"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// runCreateStaff is the first-run setup flow.
//
// There is no default admin account and no seeded password (99 §7): a shipped
// credential is a credential that survives into production. Instead the
// operator runs this once, on the box, and types a password that is never
// written to a file, a migration or the shell history.
//
//	api create-staff --email ven@evermore.co.id --role admin
func runCreateStaff(ctx context.Context, gdb *gorm.DB, log *slog.Logger) error {
	var email, role, name, kitchenCode string
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--email":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--kitchen":
			if i+1 < len(args) {
				kitchenCode = args[i+1]
				i++
			}
		}
	}

	if email == "" || role == "" {
		return errors.New("usage: api create-staff --email <address> --role <admin|staff|finance|kitchen|courier> [--name <full name>] [--kitchen <code>]")
	}
	cleanEmail, err := sanitize.Email("email", email, 254)
	if err != nil {
		return err
	}
	cleanRole, err := sanitize.Enum("role", role, "admin", "staff", "finance", "kitchen", "courier")
	if err != nil {
		return err
	}
	if name == "" {
		name = cleanEmail
	}
	cleanName, err := sanitize.Required("name", name, 120)
	if err != nil {
		return err
	}

	password, err := readPassword()
	if err != nil {
		return err
	}
	if err := security.ValidatePassword(password); err != nil {
		return err
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}

	userID := uuid.Must(uuid.NewV7())
	err = gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO app_user (id, email, password_hash, full_name, is_active, is_staff, email_verified_at)
			VALUES (?,?,?,?,TRUE,TRUE, now())`,
			userID, cleanEmail, hash, cleanName).Error; err != nil {
			return err
		}
		res := tx.Exec(`
			INSERT INTO user_role (user_id, role_id)
			SELECT ?, id FROM role WHERE code = ?`, userID, cleanRole)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("role %q does not exist", cleanRole)
		}

		// A kitchen-scoped staff profile exists even when unscoped, so the
		// repository filter has a row to read rather than a NULL join.
		var kitchenID *uuid.UUID
		if kitchenCode != "" {
			var found []uuid.UUID
			if err := tx.Raw(`SELECT id FROM kitchen WHERE code = ?`, kitchenCode).Scan(&found).Error; err != nil {
				return err
			}
			if len(found) == 0 {
				return fmt.Errorf("kitchen %q does not exist", kitchenCode)
			}
			kitchenID = &found[0]
		}
		return tx.Exec(`INSERT INTO staff_profile (user_id, kitchen_id) VALUES (?,?)`,
			userID, kitchenID).Error
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return fmt.Errorf("a user with the address %s already exists", cleanEmail)
		}
		return err
	}

	// The audit log records who was created, by the operator on the console.
	_ = gdb.WithContext(ctx).Exec(`
		INSERT INTO audit_log (id, actor_id, actor_email, action, entity_type, entity_id, after_state)
		VALUES (?,?,?,?,?,?,?::jsonb)`,
		uuid.Must(uuid.NewV7()), userID, cleanEmail, "staff.create", "app_user", userID,
		fmt.Sprintf(`{"role":%q,"created_via":"console"}`, cleanRole)).Error

	log.Info("staff user created", "email", cleanEmail, "role", cleanRole, "user_id", userID)
	if cleanRole == "admin" || cleanRole == "finance" || cleanRole == "staff" {
		fmt.Println("\nThis role requires 2FA (docs/03 Q-16). Enrol TOTP at first sign-in.")
	}
	return nil
}

// readPassword takes the password from the terminal without echoing it, so it
// never reaches the shell history or a process listing.
func readPassword() (string, error) {
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		fmt.Print("Password: ")
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		fmt.Print("Confirm:  ")
		c, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		if string(b) != string(c) {
			return "", errors.New("the two passwords do not match")
		}
		return string(b), nil
	}

	// Non-interactive: read one line from stdin, so the deployment handbook can
	// pipe from a secret store rather than typing.
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("no password on stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
