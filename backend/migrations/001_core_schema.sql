-- Parents
CREATE TABLE users (
    id          TEXT PRIMARY KEY,  -- ULID
    google_id   TEXT UNIQUE NOT NULL,
    email       TEXT NOT NULL,
    name        TEXT NOT NULL,
    timezone    TEXT,                  -- IANA timezone (e.g., "America/New_York"), updated on every API call via X-Timezone header
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sentinel user for FK integrity after account deletion.
-- Pre-seeded at database initialization. logged_by/updated_by are set to this value
-- when a user account is deleted (see §2.2 account deletion ordering).
INSERT INTO users (id, google_id, email, name) VALUES ('deleted_user', '__sentinel__', '', 'Deleted User');

-- Babies
CREATE TABLE babies (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    sex             TEXT NOT NULL CHECK (sex IN ('male', 'female')),
    date_of_birth   DATE NOT NULL,
    diagnosis_date  DATE,
    kasai_date      DATE,
    default_cal_per_feed REAL DEFAULT 67,  -- default kcal estimate for breast-direct feeds without volume
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Parent ↔ Baby link
CREATE TABLE baby_parents (
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT DEFAULT 'parent',
    joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (baby_id, user_id)
);

-- Invite codes (one active code per baby; generating a new code hard-deletes ALL prior codes for that baby)
-- A cron job periodically deletes ALL codes older than 24 hours (both used and unused).
-- Codes are 6-digit numeric strings (e.g., "483921")
-- All failure cases (expired, used, invalidated, nonexistent, race condition) return a generic "invalid or expired code" error
CREATE TABLE invites (
    code        TEXT PRIMARY KEY,  -- 6-digit numeric string
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE,
    created_by  TEXT REFERENCES users(id),  -- no ON DELETE CASCADE; app explicitly deletes invites in account deletion step 3
    used_by     TEXT REFERENCES users(id),
    used_at     DATETIME,              -- set when code is redeemed
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sessions (server-side, survives restarts)
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,  -- ULID; this is the session cookie value
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,  -- random secret used only for CSRF token derivation via HMAC-SHA256; NOT the cookie value
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Push subscriptions (per device)
CREATE TABLE push_subscriptions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    endpoint    TEXT NOT NULL UNIQUE,
    p256dh      TEXT NOT NULL,
    auth        TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
