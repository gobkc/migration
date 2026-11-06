# CHANGELOG

## v1.5.0

### Features

- Added `dialect.Locker` and implemented database-native migration locks.
- PostgreSQL uses `pg_advisory_lock`, scoped to the current database.
- MySQL uses `GET_LOCK`, scoped by a hash of the current database name.

### Fixes

- Serialized concurrent `Up` calls, including holding and final migrations, instead of racing on the `migrations` table lock row.
- Acquired the lock before `ensureTable`, so legacy `migrations` tables without the newer columns no longer bypass locking.
- Ran holding migrations before `getApplied` and `hasMigrationBeyondBaseline`, so legacy schemas are upgraded before their rows are read.
- Removed the `version = -1` lock row mechanism.

### Breaking

- Custom dialects must implement `dialect.Locker`; `Up` returns `ErrLockUnsupported` otherwise.
