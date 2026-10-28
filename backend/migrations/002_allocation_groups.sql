-- Only staff set this key, so changing a profile business name cannot reset a shared quota.
ALTER TABLE users ADD COLUMN IF NOT EXISTS allocation_group text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS users_allocation_group_idx ON users(allocation_group);
