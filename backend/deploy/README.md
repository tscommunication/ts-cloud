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
