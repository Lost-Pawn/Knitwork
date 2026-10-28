CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE TABLE IF NOT EXISTS users (
 id bigserial PRIMARY KEY, name text NOT NULL, email text UNIQUE NOT NULL,
 password_hash text NOT NULL, role text NOT NULL DEFAULT 'borrower' CHECK(role IN ('borrower','owner','manager','admin')),
 verified boolean NOT NULL DEFAULT false, suspended boolean NOT NULL DEFAULT false,
 business text NOT NULL DEFAULT '', locale text NOT NULL DEFAULT 'en', qualifications text[] NOT NULL DEFAULT '{}',
 verification_note text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sessions(token_hash text PRIMARY KEY,user_id bigint REFERENCES users(id) ON DELETE CASCADE,expires_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS resources (
 id bigserial PRIMARY KEY, owner_id bigint NOT NULL REFERENCES users(id), name text NOT NULL, category text NOT NULL,
 description text NOT NULL, location text NOT NULL, rules text NOT NULL,
 training text NOT NULL DEFAULT '', operator_policy text NOT NULL CHECK(operator_policy IN ('none','optional','required')),
 hourly_rate integer NOT NULL DEFAULT 0 CHECK(hourly_rate>=0), deposit integer NOT NULL DEFAULT 0 CHECK(deposit>=0),
 max_hours integer NOT NULL DEFAULT 4 CHECK(max_hours BETWEEN 1 AND 12), weekly_hours integer NOT NULL DEFAULT 8 CHECK(weekly_hours BETWEEN 1 AND 60),
 open_hour integer NOT NULL DEFAULT 9, close_hour integer NOT NULL DEFAULT 18,
 buffer_minutes integer NOT NULL DEFAULT 30 CHECK(buffer_minutes BETWEEN 0 AND 180),
 available_from date NOT NULL DEFAULT CURRENT_DATE, available_to date NOT NULL DEFAULT (CURRENT_DATE+365),
 maintenance boolean NOT NULL DEFAULT false, maintenance_reason text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(open_hour>=0 AND close_hour<=24 AND open_hour<close_hour), CHECK(available_to>=available_from)
);
CREATE TABLE IF NOT EXISTS operators(id bigserial PRIMARY KEY,name text NOT NULL,qualifications text[] NOT NULL DEFAULT '{}',active boolean NOT NULL DEFAULT true,open_hour integer NOT NULL DEFAULT 9,close_hour integer NOT NULL DEFAULT 18);
CREATE TABLE IF NOT EXISTS bookings (
 id bigserial PRIMARY KEY, resource_id bigint NOT NULL REFERENCES resources(id), borrower_id bigint NOT NULL REFERENCES users(id),
 operator_id bigint REFERENCES operators(id), starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL,busy_until timestamptz NOT NULL,
 status text NOT NULL DEFAULT 'requested' CHECK(status IN ('requested','waitlisted','approved','in_use','return_pending','completed','rejected','cancelled')),
 purpose text NOT NULL, assisted boolean NOT NULL DEFAULT false, estimate integer NOT NULL DEFAULT 0,
 decision_reason text NOT NULL DEFAULT '', extension_end timestamptz, extension_reason text NOT NULL DEFAULT '',
 payment_note text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), checked_in_at timestamptz, returned_at timestamptz,
 CHECK(ends_at>starts_at), CHECK(busy_until>=ends_at),
 EXCLUDE USING gist(resource_id WITH =,tstzrange(starts_at,busy_until,'[)') WITH &&) WHERE(status IN ('approved','in_use','return_pending')),
 EXCLUDE USING gist(operator_id WITH =,tstzrange(starts_at,busy_until,'[)') WITH &&) WHERE(status IN ('approved','in_use','return_pending'))
);
CREATE TABLE IF NOT EXISTS conditions(id bigserial PRIMARY KEY,booking_id bigint NOT NULL REFERENCES bookings(id),actor_id bigint REFERENCES users(id),phase text NOT NULL CHECK(phase IN ('before','after')),note text NOT NULL,damaged boolean NOT NULL DEFAULT false,mime text NOT NULL,image bytea NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),UNIQUE(booking_id,phase));
CREATE TABLE IF NOT EXISTS incidents(id bigserial PRIMARY KEY,booking_id bigint NOT NULL REFERENCES bookings(id),reporter_id bigint NOT NULL REFERENCES users(id),kind text NOT NULL CHECK(kind IN ('damage','delay','conflict','other')),description text NOT NULL,status text NOT NULL DEFAULT 'open' CHECK(status IN ('open','escalated','resolved')),resolution text NOT NULL DEFAULT '',created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS audit(id bigserial PRIMARY KEY,actor_id bigint REFERENCES users(id),booking_id bigint REFERENCES bookings(id),resource_id bigint REFERENCES resources(id),action text NOT NULL,detail text NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS notifications(id bigserial PRIMARY KEY,user_id bigint NOT NULL REFERENCES users(id),booking_id bigint REFERENCES bookings(id),kind text NOT NULL,message text NOT NULL,dedupe_key text UNIQUE,read boolean NOT NULL DEFAULT false,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS help_docs(id bigserial PRIMARY KEY,title text UNIQUE NOT NULL,body text NOT NULL);
CREATE INDEX IF NOT EXISTS booking_borrower_idx ON bookings(borrower_id,starts_at);
CREATE INDEX IF NOT EXISTS notifications_user_idx ON notifications(user_id,created_at DESC);
CREATE INDEX IF NOT EXISTS audit_booking_idx ON audit(booking_id,id);
CREATE INDEX IF NOT EXISTS help_search_idx ON help_docs USING gin(to_tsvector('english',title||' '||body));
