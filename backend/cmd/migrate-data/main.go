package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type tableCopy struct {
	name string
	copy func(*gorm.DB, *gorm.DB) (int64, error)
}

func main() {
	sqlitePath := flag.String("sqlite", "", "source SQLite database path")
	postgresDSN := flag.String("postgres", os.Getenv("DATABASE_URL"), "target PostgreSQL DSN (defaults to DATABASE_URL)")
	execute := flag.Bool("execute", false, "perform migration; without this flag only validates configuration")
	flag.Parse()
	if *sqlitePath == "" || *postgresDSN == "" {
		log.Fatal("--sqlite and --postgres are required")
	}
	absPath, err := filepath.Abs(*sqlitePath)
	if err != nil {
		log.Fatal(err)
	}
	source, err := gorm.Open(sqlite.Open("file:"+absPath+"?mode=ro"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	target, err := gorm.Open(postgres.Open(*postgresDSN), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := verifySource(source); err != nil {
		log.Fatal(err)
	}
	if err := requireEmptyTarget(target); err != nil {
		log.Fatal(err)
	}
	if !*execute {
		log.Printf("Dry run passed: source=%s is readable and PostgreSQL target is empty; add --execute to copy data", absPath)
		return
	}

	tables := []tableCopy{
		copyTable[models.User]("users"), copyTable[models.Customer]("customers"), copyTable[models.Package]("packages"),
		copyTable[models.Subscription]("subscriptions"), copyTable[models.Invoice]("invoices"), copyTable[models.Payment]("payments"),
		copyTable[models.BillingRun]("billing_runs"), copyTable[models.BillingRunItem]("billing_run_items"),
		copyTable[models.FTPServer]("ftp_servers"), copyTable[models.FTPUser]("ftp_users"),
		copyTable[models.FTPLoginLog]("ftp_login_logs"), copyTable[models.FTPTransferLog]("ftp_transfer_logs"),
		copyTable[models.SystemLogOffset]("system_log_offsets"),
	}
	err = target.Transaction(func(tx *gorm.DB) error {
		if err := database.Migrate(tx); err != nil {
			return err
		}
		for _, table := range tables {
			count, err := table.copy(source, tx)
			if err != nil {
				return fmt.Errorf("copy %s: %w", table.name, err)
			}
			log.Printf("Copied %s: %d row(s)", table.name, count)
			if count > 0 {
				if err := tx.Exec(fmt.Sprintf("SELECT setval(pg_get_serial_sequence('%s','id'), COALESCE((SELECT MAX(id) FROM %s), 1), true)", table.name, table.name)).Error; err != nil {
					return fmt.Errorf("sequence %s: %w", table.name, err)
				}
			}
		}
		return verifyForeignKeys(tx)
	})
	if err != nil {
		log.Fatalf("migration rolled back: %v", err)
	}
	log.Println("Data migration and row-count verification completed")
}

func verifySource(source *gorm.DB) error {
	var tables []string
	if err := source.Raw("SELECT name FROM sqlite_master WHERE type = 'table'").Scan(&tables).Error; err != nil {
		return fmt.Errorf("inspect SQLite source: %w", err)
	}
	for _, required := range []string{"users", "customers", "packages", "subscriptions", "invoices", "payments"} {
		found := false
		for _, table := range tables {
			if table == required {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("SQLite source is missing required table %q", required)
		}
	}
	return nil
}

func requireEmptyTarget(target *gorm.DB) error {
	var tables []string
	if err := target.Raw("SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = current_schema()").Scan(&tables).Error; err != nil {
		return fmt.Errorf("inspect PostgreSQL target: %w", err)
	}
	if len(tables) > 0 {
		return fmt.Errorf("PostgreSQL target schema must be empty; found: %s", strings.Join(tables, ", "))
	}
	return nil
}

func verifyForeignKeys(target *gorm.DB) error {
	var violations int64
	query := `SELECT COUNT(*) FROM (
		SELECT 1 FROM subscriptions s LEFT JOIN customers c ON c.id=s.customer_id LEFT JOIN packages p ON p.id=s.package_id WHERE c.id IS NULL OR p.id IS NULL
		UNION ALL SELECT 1 FROM invoices i LEFT JOIN subscriptions s ON s.id=i.subscription_id LEFT JOIN customers c ON c.id=i.customer_id LEFT JOIN packages p ON p.id=i.package_id WHERE s.id IS NULL OR c.id IS NULL OR p.id IS NULL
		UNION ALL SELECT 1 FROM payments p LEFT JOIN invoices i ON i.id=p.invoice_id LEFT JOIN customers c ON c.id=p.customer_id LEFT JOIN subscriptions s ON s.id=p.subscription_id WHERE i.id IS NULL OR c.id IS NULL OR s.id IS NULL
		UNION ALL SELECT 1 FROM billing_run_items b LEFT JOIN billing_runs r ON r.id=b.billing_run_id LEFT JOIN subscriptions s ON s.id=b.subscription_id WHERE r.id IS NULL OR s.id IS NULL
		UNION ALL SELECT 1 FROM ftp_users f LEFT JOIN subscriptions s ON s.id=f.subscription_id LEFT JOIN ftp_servers fs ON fs.id=f.ftp_server_id WHERE s.id IS NULL OR fs.id IS NULL
	) violations`
	if err := target.Raw(query).Scan(&violations).Error; err != nil {
		return fmt.Errorf("verify foreign keys: %w", err)
	}
	if violations != 0 {
		return fmt.Errorf("foreign-key verification failed: %d orphan row(s)", violations)
	}
	return nil
}

func copyTable[T any](name string) tableCopy {
	return tableCopy{name: name, copy: func(source, target *gorm.DB) (int64, error) {
		var rows []T
		if err := source.Unscoped().Find(&rows).Error; err != nil {
			return 0, err
		}
		if len(rows) == 0 {
			return 0, nil
		}
		err := target.Omit(clause.Associations).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 200).Error
		if err != nil {
			return 0, err
		}
		var targetCount int64
		if err := target.Unscoped().Model(new(T)).Count(&targetCount).Error; err != nil {
			return 0, err
		}
		if targetCount != int64(len(rows)) {
			return 0, fmt.Errorf("row-count mismatch: source=%d target=%d", len(rows), targetCount)
		}
		return int64(len(rows)), nil
	}}
}
