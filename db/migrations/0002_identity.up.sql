-- 0002 — identity, roles, permissions, sessions.
--
-- Deny-by-default authorization (99 §7): a permission the route does not
-- declare is a permission nobody has. Roles are rows, not an enum, because
-- PROMPT §3 names six and admin will want a seventh.

CREATE TABLE app_user (
  id                 UUID PRIMARY KEY,
  email              CITEXT NOT NULL UNIQUE,
  password_hash      TEXT NOT NULL,
  phone              TEXT,
  full_name          TEXT NOT NULL,
  is_active          BOOLEAN NOT NULL DEFAULT TRUE,
  is_staff           BOOLEAN NOT NULL DEFAULT FALSE,
  email_verified_at  TIMESTAMPTZ,
  phone_verified_at  TIMESTAMPTZ,
  last_login_at      TIMESTAMPTZ,
  failed_attempts    INT NOT NULL DEFAULT 0,
  locked_until       TIMESTAMPTZ,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT app_user_phone_format CHECK (phone IS NULL OR phone ~ '^\+?[0-9]{8,20}$')
);
CREATE INDEX app_user_staff_idx ON app_user (is_staff) WHERE is_staff;

CREATE TABLE role (
  id          UUID PRIMARY KEY,
  code        TEXT NOT NULL UNIQUE,
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_staff    BOOLEAN NOT NULL DEFAULT TRUE,
  is_system   BOOLEAN NOT NULL DEFAULT FALSE,
  requires_2fa BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permission (
  id          UUID PRIMARY KEY,
  code        TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE role_permission (
  role_id       UUID NOT NULL REFERENCES role(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permission(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_role (
  user_id     UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  role_id     UUID NOT NULL REFERENCES role(id) ON DELETE RESTRICT,
  granted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  granted_by  UUID REFERENCES app_user(id),
  PRIMARY KEY (user_id, role_id)
);

-- Staff profile carries the kitchen scope (docs/02 D-21). NULL = all kitchens.
-- The FK is added in 0006 once the kitchen table exists.
CREATE TABLE staff_profile (
  user_id     UUID PRIMARY KEY REFERENCES app_user(id) ON DELETE CASCADE,
  kitchen_id  UUID,
  employee_no TEXT,
  notes       TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- TOTP secrets are stored encrypted at the application edge; the column holds
-- ciphertext, never a base32 secret in the clear.
CREATE TABLE user_totp (
  user_id       UUID PRIMARY KEY REFERENCES app_user(id) ON DELETE CASCADE,
  secret_cipher TEXT NOT NULL,
  confirmed_at  TIMESTAMPTZ,
  last_used_step BIGINT,
  recovery_codes JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Refresh tokens are stored hashed and are revocable; a jti denylist on logout
-- is the rotation story (99 §7).
CREATE TABLE refresh_token (
  id            UUID PRIMARY KEY,
  user_id       UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  token_hash    TEXT NOT NULL UNIQUE,
  parent_id     UUID REFERENCES refresh_token(id) ON DELETE SET NULL,
  user_agent    TEXT,
  ip            INET,
  issued_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at    TIMESTAMPTZ NOT NULL,
  revoked_at    TIMESTAMPTZ,
  revoked_reason TEXT
);
CREATE INDEX refresh_token_user_idx ON refresh_token (user_id, expires_at DESC);
CREATE INDEX refresh_token_live_idx ON refresh_token (expires_at) WHERE revoked_at IS NULL;

-- Single-use, hashed, short-lived, attempt-capped (99 §7).
CREATE TABLE verification_token (
  id          UUID PRIMARY KEY,
  user_id     UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  purpose     TEXT NOT NULL CHECK (purpose IN ('EMAIL_VERIFY','PASSWORD_RESET','PHONE_OTP')),
  token_hash  TEXT NOT NULL,
  attempts    INT NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at  TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ
);
CREATE INDEX verification_token_lookup_idx ON verification_token (token_hash) WHERE consumed_at IS NULL;
CREATE INDEX verification_token_user_idx ON verification_token (user_id, purpose, created_at DESC);

CREATE TRIGGER app_user_touch BEFORE UPDATE ON app_user
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER role_touch BEFORE UPDATE ON role
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER staff_profile_touch BEFORE UPDATE ON staff_profile
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER user_totp_touch BEFORE UPDATE ON user_totp
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
