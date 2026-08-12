-- 0001 — extensions and the platform tables.
--
-- Nothing business-specific lives here: parameters, audit, idempotency, jobs and
-- the notification log are cross-cutting and portable (CLAUDE.md §2).
--
-- The extensions must already exist; they need superuser and are created by
-- docs/RUN-WHEN-BACK.md §4 before the first migration runs. This file asserts
-- they are present rather than creating them, so a missing extension fails here
-- loudly instead of at 0006 with a confusing PostGIS error.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'btree_gist') THEN
    RAISE EXCEPTION 'extension btree_gist is required (price overlap constraints) — see docs/RUN-WHEN-BACK.md §4';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'postgis') THEN
    RAISE EXCEPTION 'extension postgis is required (kitchen routing) — see docs/RUN-WHEN-BACK.md §4';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'citext') THEN
    RAISE EXCEPTION 'extension citext is required (case-insensitive email) — see docs/RUN-WHEN-BACK.md §4';
  END IF;
END $$;

-- ── sys_parameters ───────────────────────────────────────────────────────────
-- Every value the business might change without a deploy (CLAUDE.md §7).
-- value_type drives both the admin form control and server-side parsing.
CREATE TABLE sys_parameters (
  id            UUID PRIMARY KEY,
  key           TEXT NOT NULL UNIQUE,
  value         TEXT NOT NULL,
  value_type    TEXT NOT NULL
                CHECK (value_type IN ('string','int','bool','money','bps','time','date','json','duration')),
  param_group   TEXT NOT NULL DEFAULT 'general',
  label         TEXT NOT NULL,
  description   TEXT NOT NULL DEFAULT '',
  is_secret     BOOLEAN NOT NULL DEFAULT FALSE,
  is_system     BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order    INT NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by    UUID
);
CREATE INDEX sys_parameters_group_idx ON sys_parameters (param_group, sort_order);
COMMENT ON TABLE sys_parameters IS
  'Runtime configuration. If Steven might change it without a deploy, it belongs here (CLAUDE.md §7).';

-- ── audit_log ────────────────────────────────────────────────────────────────
-- Append-only. Every staff action touching money, prices, customer type,
-- credits, package expiry or kitchen assignment writes one row (PROMPT §3).
CREATE TABLE audit_log (
  id            UUID PRIMARY KEY,
  actor_id      UUID,
  actor_email   TEXT,
  action        TEXT NOT NULL,
  entity_type   TEXT NOT NULL,
  entity_id     UUID,
  before_state  JSONB,
  after_state   JSONB,
  reason        TEXT,
  ip            INET,
  user_agent    TEXT,
  occurred_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_entity_idx ON audit_log (entity_type, entity_id, occurred_at DESC);
CREATE INDEX audit_log_actor_idx  ON audit_log (actor_id, occurred_at DESC);
CREATE INDEX audit_log_action_idx ON audit_log (action, occurred_at DESC);

-- The database refuses to rewrite history, not just the application
-- (CLAUDE.md §4: "the database enforces the invariant").
CREATE OR REPLACE FUNCTION reject_mutation() RETURNS TRIGGER AS $$
BEGIN
  RAISE EXCEPTION '% is append-only: % is not permitted', TG_TABLE_NAME, TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_append_only
  BEFORE UPDATE OR DELETE ON audit_log
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();

-- ── idempotency ──────────────────────────────────────────────────────────────
-- In Postgres, not Redis (docs/02 D-4): a key that guards an order which
-- creates money must survive a cache restart and commit in the same
-- transaction as the write it protects.
CREATE TABLE idempotency_key (
  id             UUID PRIMARY KEY,
  key            TEXT NOT NULL,
  user_id        UUID,
  endpoint       TEXT NOT NULL,
  request_hash   TEXT NOT NULL,
  response_code  INT,
  response_body  JSONB,
  state          TEXT NOT NULL DEFAULT 'IN_PROGRESS'
                 CHECK (state IN ('IN_PROGRESS','COMPLETED')),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at   TIMESTAMPTZ,
  expires_at     TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '24 hours',
  UNIQUE (key, endpoint)
);
CREATE INDEX idempotency_key_expiry_idx ON idempotency_key (expires_at);

-- ── job queue ────────────────────────────────────────────────────────────────
-- Postgres-backed so a job is enqueued in the same transaction as the state
-- change that caused it; Redis carries cache and rate limits only.
CREATE TABLE job (
  id            UUID PRIMARY KEY,
  kind          TEXT NOT NULL,
  payload       JSONB NOT NULL DEFAULT '{}'::jsonb,
  run_after     TIMESTAMPTZ NOT NULL DEFAULT now(),
  attempts      INT NOT NULL DEFAULT 0,
  max_attempts  INT NOT NULL DEFAULT 5,
  last_error    TEXT,
  state         TEXT NOT NULL DEFAULT 'PENDING'
                CHECK (state IN ('PENDING','RUNNING','DONE','FAILED')),
  dedupe_key    TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX job_claim_idx ON job (state, run_after) WHERE state = 'PENDING';
-- One pending job per dedupe key: re-running the expiry sweep must not enqueue
-- a second copy of the same reminder.
CREATE UNIQUE INDEX job_dedupe_idx ON job (dedupe_key)
  WHERE dedupe_key IS NOT NULL AND state IN ('PENDING','RUNNING');

-- ── notification_log ─────────────────────────────────────────────────────────
-- Support has to be able to prove what was sent (PROMPT §11).
CREATE TABLE notification_log (
  id            UUID PRIMARY KEY,
  customer_id   UUID,
  channel       TEXT NOT NULL CHECK (channel IN ('EMAIL','WHATSAPP')),
  template      TEXT NOT NULL,
  recipient     TEXT NOT NULL,
  subject       TEXT,
  locale        TEXT NOT NULL DEFAULT 'id-ID',
  state         TEXT NOT NULL DEFAULT 'QUEUED'
                CHECK (state IN ('QUEUED','SENT','FAILED')),
  provider      TEXT,
  provider_ref  TEXT,
  error         TEXT,
  reference_type TEXT,
  reference_id  UUID,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  sent_at       TIMESTAMPTZ
);
CREATE INDEX notification_log_customer_idx ON notification_log (customer_id, created_at DESC);
CREATE INDEX notification_log_ref_idx ON notification_log (reference_type, reference_id);

-- ── updated_at trigger, used by every mutable table ──────────────────────────
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sys_parameters_touch BEFORE UPDATE ON sys_parameters
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER job_touch BEFORE UPDATE ON job
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
