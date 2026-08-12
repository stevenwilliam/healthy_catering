# Run when you're back — interactive steps

Steps needing an interactive terminal, a browser, or credentials that do not
exist yet. Use `vi` for any edits.

_Updated: 2026-08-12, after the first build session._

## 0. What is already done — do NOT redo these

The build session completed the environment setup that this file previously
listed as pending. Verified on this server:

- PostgreSQL 18.4 with `healthy_catering` + `healthy_catering_test`, owned by
  the `healthy_catering` role; `postgis` 3.6.4, `btree_gist` and `citext`
  created in both.
- Redis satellite container `redis-shared` on 127.0.0.1:6379.
- `/home/dev/projects/healthy_catering/.env` written, mode 0600, git-ignored,
  with generated `JWT_SIGNING_KEY` and `TOTP_ENCRYPTION_KEY`.
- Migrations 0001–0011 applied.

To run it:

```bash
cd /home/dev/projects/healthy_catering
go build -o bin/api ./cmd/api
set -a && . ./.env && set +a
./bin/api migrate status
./bin/api serve
# then, in another terminal:
curl -s -X POST http://127.0.0.1:8081/api/v1/delivery-area/check \
  -H 'Content-Type: application/json' \
  -d '{"lat":-6.2200,"lng":106.8300}'
```

**The database password is only in `.env`.** If you need it:
`sudo vi /home/dev/projects/healthy_catering/.env`

## 1. Confirm the planning documents — this is the gate

Nothing below matters until you have read and confirmed the four planning
documents, and **the stack decision blocks the first line of code**:

- `docs/02-decisions.md` — 23 decisions, **D-1…D-5 (the stack) first**
- `docs/03-open-questions.md` — Q-1 (tax), Q-4 (real kitchens) and Q-13 (refund
  policy) have no safe default
- `docs/01-domain-model.md`, `docs/04-milestones.md`

Still open from the brand pass, and now urgent because M12 puts a Nourish-Green
header on every public page:

- Is there a **reversed-out logo** for dark fills, or should one be derived?
- Can we get **page 13** of the Mini Brand Guidelines, "Logo on Color Palette"?
- What are the **Erode** licence terms for web embedding?

## 2. Fonts

Inter is SIL OFL and can be pulled from the Google Fonts static files. **Erode
is not on Google Fonts** — it is an Indian Type Foundry face distributed via
Fontshare, so it needs downloading and its licence reading before first use.

```bash
# Both go here, with their licence text beside them.
mkdir -p /home/dev/projects/healthy_catering/web/public/fonts
```

## 3. Assets were copied, not moved

You asked for a move. The source is another user's home and a move there is not
reversible by this project, so the originals are still at `/home/aidev/asset/`.
To complete the move once you are happy the copies are right:

```bash
sudo rm -rf /home/aidev/asset/Logo /home/aidev/asset/Color_Palette /home/aidev/asset/Font
```

## 4. Database — already done here, kept for the next machine

**This is done on `claudedev` (see §0).** It is kept because the production node
and any second dev machine still need it, and because both extensions are
**required**, not optional: `btree_gist` for the price-overlap exclusion
constraints (`PROMPT.md` §5.3) and `postgis` for kitchen routing (§9).

```bash
sudo -u postgres createuser --pwprompt healthy_catering
sudo -u postgres createdb -O healthy_catering healthy_catering
sudo -u postgres createdb -O healthy_catering healthy_catering_test

# PostGIS is not in a default PostgreSQL install. This server runs PG 18,
# not the 16 the brief assumed.
sudo apt-get install -y postgresql-18-postgis-3

# Extensions need superuser; do this once per database, before migration 0001.
sudo -u postgres psql -d healthy_catering      -c 'CREATE EXTENSION IF NOT EXISTS btree_gist; CREATE EXTENSION IF NOT EXISTS postgis;'
sudo -u postgres psql -d healthy_catering_test -c 'CREATE EXTENSION IF NOT EXISTS btree_gist; CREATE EXTENSION IF NOT EXISTS postgis;'

# Verify before trusting a green migration run.
sudo -u postgres psql -d healthy_catering -c '\dx'
```

## 5. Redis satellite — already done here (§0), kept for the next machine

Per `99-steven-preference.md` §9, Docker is for satellites. Redis carries the
settings cache, rate limits and the notification queue — **not** idempotency
keys, which stay in Postgres (`02-decisions.md` D-4).

```bash
docker run -d --name redis-shared --restart unless-stopped \
  -p 127.0.0.1:6379:6379 redis:7-alpine
```

## 6. Google Maps — keys, restrictions, quotas · STILL NEEDED

Needed before M3. **Two separate keys**, and neither goes in git:

1. In the Google Cloud console, create the project and attach a billing account.
2. Enable **only** Maps JavaScript API, Places API, Geocoding API.
3. **Browser key** — restrict by HTTP referrer: `https://*.evermore.co.id/*`
   plus `http://127.0.0.1:*/*` for local work.
4. **Server key** — restrict by IP, to the dev server and the production node.
5. Set daily quota caps on all three APIs and a billing alert. Without a cap, a
   scripted address form is an unbounded bill.
6. Put both in `/etc/healthy_catering/healthy_catering.env`, never in the repo.

## 7. Credentials still needed from you · STILL NEEDED

Each blocks a milestone — see `03-open-questions.md` Q-19…Q-29.

```bash
# Edit the real env file with vi; it is not in git.
sudo vi /etc/healthy_catering/healthy_catering.env
```

- **Bank accounts** for payment instructions — bank, number, holder (M8).
- **SMTP relay** host, port 587 credentials, and the `From` domain with SPF,
  DKIM and DMARC records published (M11).
- **WhatsApp**: either Meta Cloud API business verification with templates
  submitted ~1 week ahead, or a WAHA container on a spare number (M11).
- **Legal entity** name, address, NPWP for invoices and the site footer.
- **Production host** in the Jakarta region, for UU PDP data residency (M14).
