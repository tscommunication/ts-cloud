package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	if !*execute {
		log.Printf("Dry run: source=%s target configured; add --execute to copy data", absPath)
		return
	}
	source, err := gorm.Open(sqlite.Open("file:"+absPath+"?mode=ro"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	target, err := gorm.Open(postgres.Open(*postgresDSN), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(target); err != nil {
		log.Fatal(err)
	}

	tables := []tableCopy{
		copyTable[models.User]("users"), copyTable[models.Customer]("customers"), copyTable[models.Package]("packages"),
		copyTable[models.Subscription]("subscriptions"), copyTable[models.Invoice]("invoices"), copyTable[models.Payment]("payments"),
		copyTable[models.BillingRun]("billing_runs"), copyTable[models.BillingRunItem]("billing_run_items"),
		copyTable[models.FTPServer]("ftp_servers"), copyTable[models.FTPUser]("ftp_users"),
		copyTable[models.FTPLoginLog]("ftp_login_logs"), copyTable[models.FTPTransferLog]("ftp_transfer_logs"),
		copyTable[models.SystemLogOffset]("system_log_offsets"),
	}
	for _, table := range tables {
		count, err := table.copy(source, target)
		if err != nil {
			log.Fatalf("copy %s: %v", table.name, err)
		}
		log.Printf("Copied %s: %d row(s)", table.name, count)
		if count > 0 {
			if err := target.Exec(fmt.Sprintf("SELECT setval(pg_get_serial_sequence('%s','id'), COALESCE((SELECT MAX(id) FROM %s), 1), true)", table.name, table.name)).Error; err != nil {
				log.Fatalf("sequence %s: %v", table.name, err)
			}
		}
	}
	log.Println("Data migration and row-count verification completed")
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
