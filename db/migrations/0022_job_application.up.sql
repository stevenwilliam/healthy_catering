-- Career applications submitted from the public site.
--
-- Steven, 2026-08-18: the Career page is a web form, its input is sanitised,
-- and it accepts NO FILES.
--
-- No attachment column, deliberately and permanently. An upload endpoint on an
-- unauthenticated public page is the highest-risk surface a marketing site can
-- have — content-type spoofing, archive bombs, malware handed straight to
-- whoever opens it in HR, and storage that has to be scanned and expired. A CV
-- arrives by email, from a person we have already replied to.
CREATE TABLE job_application (
  id          UUID PRIMARY KEY,
  full_name   TEXT NOT NULL,
  email       CITEXT NOT NULL,
  phone       TEXT NOT NULL DEFAULT '',
  position    TEXT NOT NULL,
  message     TEXT NOT NULL DEFAULT '',
  -- Which language the applicant was reading. Tells us who to reply in.
  locale      TEXT NOT NULL DEFAULT 'id' CHECK (locale IN ('id','en','zh')),
  status      TEXT NOT NULL DEFAULT 'NEW'
              CHECK (status IN ('NEW','REVIEWING','CONTACTED','REJECTED','HIRED')),
  -- For abuse tracing only. These are PERSONAL DATA under UU PDP and are part
  -- of what an erasure request has to reach (docs/12).
  submitted_ip TEXT NOT NULL DEFAULT '',
  user_agent   TEXT NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by   UUID REFERENCES app_user(id),

  CONSTRAINT job_application_name_present  CHECK (length(btrim(full_name)) > 0),
  CONSTRAINT job_application_email_present CHECK (length(btrim(email::text)) > 0)
);
CREATE INDEX job_application_queue_idx ON job_application (status, created_at DESC);
CREATE INDEX job_application_email_idx ON job_application (email);

COMMENT ON TABLE job_application IS
  'Career form submissions. No attachments by design — see migration 0022.';

CREATE TRIGGER job_application_touch BEFORE UPDATE ON job_application
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ── Open positions ──────────────────────────────────────────────────────────
--
-- Steven, 2026-08-18: the career page shows what is being hired for right now,
-- configurable from the back office.
--
-- A table rather than a rich-text block, because these rows do two jobs: they
-- are the list at the top of the page AND the options in the form's position
-- field. Free prose could do the first but not the second, and a form that
-- lets someone apply for a role that is not open is a form that generates work
-- for HR.
CREATE TABLE job_opening (
  id          UUID PRIMARY KEY,
  title       TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  -- One line under the title: location, shift, whatever matters.
  summary     TEXT NOT NULL DEFAULT '',
  sort_order  INT NOT NULL DEFAULT 0,
  is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by  UUID REFERENCES app_user(id),
  CONSTRAINT job_opening_title_present CHECK (length(btrim(title)) > 0)
);
CREATE INDEX job_opening_active_idx ON job_opening (is_active, sort_order);

CREATE TRIGGER job_opening_touch BEFORE UPDATE ON job_opening
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Starter rows, so the page is not empty on day one. Real vacancies replace
-- them from the back office.
INSERT INTO job_opening (id, title, slug, summary, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000d01','Juru Masak','juru-masak','Dapur Jakarta Selatan · shift pagi',1),
 ('00000000-0000-7000-8000-000000000d02','Staf Dapur','staf-dapur','Persiapan dan pengemasan · shift pagi',2),
 ('00000000-0000-7000-8000-000000000d03','Kurir','kurir','Bawa kendaraan sendiri · area Jakarta',3),
 ('00000000-0000-7000-8000-000000000d04','Customer Service','customer-service','Senin–Sabtu · dari kantor',4);
