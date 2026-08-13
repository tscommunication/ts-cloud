# Frontend production deployment

Run the deployment from the repository checkout on the production host:

```bash
frontend/deploy/deploy-production.sh
```

The script installs the locked dependencies, runs lint, creates a production
build, checks for high-severity dependency vulnerabilities, backs up the live
frontend, deploys with delayed updates, and verifies the public URL.

Backups are stored under:

```text
~/.local/state/ts-cloud/frontend-backups/
```

The deployment output prints the exact rollback command. It has this form:

```bash
frontend/deploy/rollback-production.sh ~/.local/state/ts-cloud/frontend-backups/<release-id>
```

Nginx configuration changes remain a separate privileged operation:

```bash
sudo cp frontend/deploy/nginx/cloud.tscommunication.com.bd.conf /etc/nginx/sites-available/ts-cloud
sudo nginx -t
sudo systemctl reload nginx
```
