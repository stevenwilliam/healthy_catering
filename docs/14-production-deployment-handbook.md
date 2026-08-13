# 14 — Production deployment handbook

**Copy-paste, from an empty Ubuntu machine, with full absolute paths.**
Editor is `vi` throughout (99 §2).
**Date:** 2026-08-13

Everything here was **run on the development server** (`claudedev`) and works;
the production-only steps — the domain, the TLS certificate, the real
credentials — are marked ⚠️ and have not been run because those inputs do not
exist yet.

---

## 0. What you need before you start

| Input | Why | State |
|---|---|---|
| A Jakarta-region Ubuntu LTS node with sudo | UU PDP data residency | ⚠️ not provisioned |
| `evermore.co.id` DNS pointing at it | TLS and OG previews | ⚠️ not pointed |
| Real bank account details | Payment instructions | ⚠️ placeholder |
| Google Maps browser + server keys | Address pin picker | ⚠️ not issued |
| SMTP relay with SPF/DKIM/DMARC | Transactional email | ⚠️ mailpit only |
| Real kitchen list with map pins | Routing | ⚠️ placeholders |

**Do not point a public domain at this until the bank account is real.** The
seeded account reads `PT EVERMORE (DUMMY — REPLACE BEFORE LAUNCH)`, and a live
site would print those instructions to customers.

---

## 1. System packages

```bash
sudo apt-get update
sudo apt-get install -y postgresql-18 postgresql-18-postgis-3 nginx git curl ca-certificates
```

Go is only needed if you build on the box. The binary is static, so building
elsewhere and copying it is also fine.

```bash
# Only if building here:
curl -fsSLO https://go.dev/dl/go1.26.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
```

## 2. Database

```bash
sudo -u postgres createuser --pwprompt evermore
sudo -u postgres createdb -O evermore evermore
sudo -u postgres createdb -O evermore evermore_test

# Extensions need superuser and must exist BEFORE the first migration.
sudo -u postgres psql -d evermore      -c 'CREATE EXTENSION IF NOT EXISTS btree_gist; CREATE EXTENSION IF NOT EXISTS postgis; CREATE EXTENSION IF NOT EXISTS citext;'
sudo -u postgres psql -d evermore_test -c 'CREATE EXTENSION IF NOT EXISTS btree_gist; CREATE EXTENSION IF NOT EXISTS postgis; CREATE EXTENSION IF NOT EXISTS citext;'

# Verify before trusting a green migration run.
sudo -u postgres psql -d evermore -c '\dx'
```

## 3. Redis satellite

```bash
sudo apt-get install -y docker.io
sudo docker run -d --name redis-shared --restart unless-stopped \
  -p 127.0.0.1:6379:6379 redis:7-alpine
sudo docker exec redis-shared redis-cli ping   # expect PONG
```

## 4. The code

```bash
sudo mkdir -p /srv/evermore && sudo chown "$USER":"$USER" /srv/evermore
git clone https://github.com/stevenwilliam/healthy_catering.git /srv/evermore
cd /srv/evermore
go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" -o /srv/evermore/bin/api ./cmd/api
/srv/evermore/bin/api version
```

## 5. Configuration and secrets

Nothing secret is in the repository. The service reads one root-owned file.

```bash
sudo mkdir -p /etc/evermore
sudo vi /etc/evermore/evermore.env
```

```ini
APP_ENV=production
APP_BIND=127.0.0.1
APP_PORT=8081
APP_BASE_URL=https://www.evermore.co.id
APP_TIMEZONE=Asia/Jakarta
LOG_LEVEL=info
CORS_ALLOWED_ORIGINS=https://www.evermore.co.id
TRUSTED_PROXIES=127.0.0.1

DATABASE_URL=postgres://evermore:REPLACE@127.0.0.1:5432/evermore?sslmode=disable
TEST_DATABASE_URL=postgres://evermore:REPLACE@127.0.0.1:5432/evermore_test?sslmode=disable

# openssl rand -base64 48
JWT_SIGNING_KEY=REPLACE
TOTP_ENCRYPTION_KEY=REPLACE

REDIS_URL=redis://127.0.0.1:6379/0
GOOGLE_MAPS_BROWSER_KEY=REPLACE
GOOGLE_MAPS_SERVER_KEY=REPLACE
TURNSTILE_SECRET=REPLACE
```

```bash
sudo chown root:evermore /etc/evermore/evermore.env
sudo chmod 640 /etc/evermore/evermore.env
```

**Production refuses to start** without both Maps keys and a CORS list — the
config validates that deliberately, so a misconfigured production node fails
loudly at boot rather than quietly at the first order.

## 6. The service

```bash
sudo cp /srv/evermore/deploy/evermore.service /etc/systemd/system/evermore.service
sudo vi /etc/systemd/system/evermore.service     # set User=, WorkingDirectory=, paths
sudo systemctl daemon-reload
sudo systemctl enable --now evermore
systemctl status evermore --no-pager
```

Migrations run in `ExecStartPre`, so **the schema is applied before the new
binary serves** — a rollout can never serve against a schema it does not
expect.

## 7. First admin account

There is no default admin and no seeded password.

```bash
cd /srv/evermore
set -a; . /etc/evermore/evermore.env; set +a
/srv/evermore/bin/api create-staff --email ven@evermore.co.id --role admin --name "Steven William"
# The password is typed, never echoed, and never reaches the shell history.
```

## 8. nginx and TLS

```bash
sudo cp /srv/evermore/deploy/nginx/evermore.conf /etc/nginx/sites-available/evermore
sudo vi /etc/nginx/sites-available/evermore   # listen 80/443, real server_name
sudo ln -sf /etc/nginx/sites-available/evermore /etc/nginx/sites-enabled/evermore
sudo nginx -t && sudo systemctl reload nginx
```

⚠️ Then, once DNS resolves:

```bash
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot --nginx -d www.evermore.co.id -d evermore.co.id
sudo certbot renew --dry-run
```

Only after the certificate exists, uncomment the `return 301 https://…` line
and the TLS server block. Doing it earlier breaks the ACME challenge.

## 9. Verify the deployment

```bash
curl -s https://www.evermore.co.id/healthz
curl -s https://www.evermore.co.id/robots.txt | head -3

# What a WhatsApp preview bot sees. Use curl, not a browser — the browser
# hides exactly what this shows (99 §13).
curl -s https://www.evermore.co.id/ | grep -i 'og:\|<title'
```

## 10. Backups

⚠️ **Not yet written, and an untested restore is not a backup.**

```bash
sudo mkdir -p /var/backups/evermore
sudo vi /usr/local/bin/evermore-backup.sh
```

```bash
#!/bin/bash
set -euo pipefail
STAMP=$(date +%Y%m%d-%H%M)
pg_dump "$DATABASE_URL" | gzip > /var/backups/evermore/evermore-$STAMP.sql.gz
find /var/backups/evermore -name '*.sql.gz' -mtime +14 -delete
# Then copy off the machine — a backup on the same disk is not a backup.
```

**Before trusting it, restore into `evermore_test` and run the security suite
against the restored copy.** That is the only thing that proves the dump is
usable.

## 11. Rollback

```bash
cd /srv/evermore
git log --oneline -5
git checkout <previous-sha>
go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" -o /srv/evermore/bin/api ./cmd/api
sudo systemctl restart evermore
```

Migrations are **forward-only in production**. Rolling the binary back is safe;
rolling the schema back is not, and `api migrate down` refuses in production
unless `ALLOW_MIGRATE_DOWN=yes` is set deliberately.

---

## 12. What is deployed on `claudedev` today

Run and verified on 2026-08-13:

| Item | Value |
|---|---|
| URL | `http://192.168.88.101:8090` (IP + own port, as Steven asked) |
| API bind | `127.0.0.1:8081` — **not reachable from the LAN**, verified |
| nginx | `/etc/nginx/sites-available/evermore`, listening on 8090 |
| systemd | `evermore.service`, enabled, restarts cleanly |
| Secrets | `/etc/evermore/evermore.env`, `root:dev`, mode 640 |
| Database | `healthy_catering` on the shared PostgreSQL 18 |

ruuma is untouched: it still owns `listen 80 default_server` and answers on the
bare IP.
