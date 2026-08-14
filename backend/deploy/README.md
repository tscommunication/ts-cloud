# Backend production deployment

Build and verify the backend before replacing the production binary:

```bash
cd backend
go test ./...
go vet ./...
go build -trimpath -o /tmp/ts-cloud-next ./cmd/server
```

Install the systemd service configuration and grant the service account
read-only access to the current VSFTPD log:

```bash
sudo setfacl -m u:tscloud:r /var/log/vsftpd.log
sudo cp deploy/ts-cloud.service /etc/systemd/system/ts-cloud.service
sudo systemctl daemon-reload
```

`SupplementaryGroups=adm` keeps log access working after logrotate creates a
new `/var/log/vsftpd.log` owned by `root:adm`. The ACL handles the current log,
which may still be owned by `nobody:nogroup`.

Deploy the verified binary with a recoverable backup:

```bash
sudo cp -a /opt/ts-cloud/backend/ts-cloud /opt/ts-cloud/backend/ts-cloud.backup
sudo install -o root -g root -m 0755 /tmp/ts-cloud-next /opt/ts-cloud/backend/ts-cloud
sudo systemctl restart ts-cloud.service
```

Verify startup and health:

```bash
systemctl is-active ts-cloud.service
curl --retry 10 --retry-connrefused --retry-delay 1 --fail http://127.0.0.1:8080/health
```

## PostgreSQL cutover

The application supports `DB_TYPE=sqlite` and `DB_TYPE=postgres`. Schema
changes are applied once and recorded in `schema_migrations`; startup no longer
runs an unversioned migration over every table.

Install PostgreSQL client tools and create a dedicated database/user. Put the
connection string in `/opt/ts-cloud/backend/.env` only after the data copy has
been verified. Do not commit credentials; use `deploy/postgres.env.example` as
the variable reference.

Build the migration utility:

```bash
cd backend
go build -trimpath -o /tmp/ts-cloud-migrate-data ./cmd/migrate-data
```

With `DATABASE_URL` exported for the empty target database, validate arguments
first (this performs no writes), then stop the API during the real copy:

```bash
/tmp/ts-cloud-migrate-data --sqlite /opt/ts-cloud/backend/ts-cloud.db
sudo systemctl stop ts-cloud.service
/tmp/ts-cloud-migrate-data --sqlite /opt/ts-cloud/backend/ts-cloud.db --execute
```

The utility creates the versioned schema, copies tables in dependency order,
preserves IDs, advances PostgreSQL sequences, and fails on a row-count mismatch.
After it succeeds, set `DB_TYPE=postgres`, restart the service, and verify
`/health`. Keep the original SQLite file read-only through the rollback window.

## Backup and restore

Install scripts and the daily timer:

```bash
sudo install -d -o root -g root -m 0750 /opt/ts-cloud/backend/deploy /var/backups/ts-cloud
sudo install -o root -g root -m 0750 deploy/backup-database.sh deploy/restore-database.sh /opt/ts-cloud/backend/deploy/
sudo install -o root -g root -m 0644 deploy/ts-cloud-backup.service deploy/ts-cloud-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now ts-cloud-backup.timer
sudo systemctl start ts-cloud-backup.service
sudo systemctl status ts-cloud-backup.service --no-pager
```

Backups are atomic, private, checksummed, and retained for 14 days by default.
Override the root or retention with `TS_CLOUD_BACKUP_ROOT` and
`TS_CLOUD_BACKUP_RETENTION_DAYS` in the backup service when required.

Restore is intentionally manual and requires an explicit confirmation flag:

```bash
sudo /opt/ts-cloud/backend/deploy/restore-database.sh /var/backups/ts-cloud/<backup-file> --confirm-restore
```

The restore script validates the checksum when present, stops the API, restores
the configured database, and starts the API again. Test restore procedures on a
separate host/database before relying on them for disaster recovery.
