```markdown
# Go Migration Library

A lightweight, production-ready Golang database migration library supporting **PostgreSQL** and **MySQL**.  
Designed for multi-Pod, cloud-native environments, **no extra lock table required**, with cross-database locking, dirty protection, checksum verification, and execution time tracking.

---

## Features

- ✅ Supports PostgreSQL and MySQL  
- ✅ Cross-database safe lock without a separate lock table  
- ✅ Dirty state protection to prevent incomplete migrations  
- ✅ File checksum verification to prevent SQL tampering  
- ✅ Supports embed.FS or custom Source  
- ✅ Execution time tracking for each migration  
- ✅ Up / Down migration support  
- ✅ Extensible Dialect for different databases  
- ✅ Suitable for multi-Pod deployments  

---

## File Naming Convention

Migration files must follow:

```

<version>*<description>.up.sql <version>*<description>.down.sql

```

Example:

```

202502021200_create_user.up.sql
202502021200_create_user.down.sql

````

- `<version>`: unique integer or timestamp  
- `<description>`: human-readable description  
- `.up.sql`: up migration  
- `.down.sql`: down migration  

---

## Installation

```bash
go get github.com/gobkc/migration
````

---

## Usage

### 1. Using embed.FS

```go
package main

import (
    "context"
    "database/sql"
    "embed"
    "log"

    "github.com/gobkc/migration"
    "github.com/gobkc/migration/dialect"
    "github.com/gobkc/migration/source"

    _ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var fs embed.FS

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }

    m := migration.New(
        db,
        dialect.Postgres{},
        source.NewEmbed(fs),
    )

    if err := m.Up(context.Background()); err != nil {
        log.Fatal(err)
    }

    log.Println("Migration completed successfully")
}
```

---

### 2. Main API

```go
type Migrator struct {
    db      *sql.DB
    dialect dialect.Dialect
    source  source.Source
}

// Create a new Migrator
func New(db *sql.DB, d dialect.Dialect, s source.Source) *Migrator

// Execute Up migrations
func (m *Migrator) Up(ctx context.Context) error
```

* `db`: database connection
* `dialect`: database dialect (Postgres / MySQL)
* `source`: migration file source (Embed / custom)

---

## Notes

1. **Dirty state protection**

   * If the last migration did not complete, `dirty=true` in `schema_migrations` will prevent further migration.
   * Requires manual repair if dirty.

2. **Checksum verification**

   * Previously applied SQL's checksum is recorded.
   * If a migration file is modified, migration will fail.

3. **Locking mechanism**

   * Uses `version = -1` as a leader row for cross-Pod locking.
   * Only one instance can execute migrations at a time.
   * Conflicts or failures will return `ErrLocked`.

4. **Multi-Pod deployment recommendation**

   * Run migrations in a separate command or initContainer.
   * Avoid running migrations automatically in the main application container.

5. **Down migrations**

   * Supported for rollback, but ensure rollback SQL is correct.

6. **Dialect extension**

   * To support other databases, implement the `Dialect` interface.

---

## Table Structure

```sql
CREATE TABLE schema_migrations (
    version BIGINT PRIMARY KEY,
    checksum VARCHAR(64) NOT NULL,
    applied_at TIMESTAMP NOT NULL,
    execution_time_s BIGINT NOT NULL,
    dirty BOOLEAN NOT NULL
);
```

* `version`: migration version
* `checksum`: SHA256 checksum of SQL file
* `applied_at`: applied timestamp
* `execution_time_s`: execution duration in milliseconds
* `dirty`: flag indicating migration not completed

> `version = -1` is used as a lock row.

---

## Recommended Usage

* Run migrations in a **separate command or initContainer**.
* Ensure only one instance performs migration in multi-Pod deployments.
* Do not call `Up` automatically at application startup to avoid conflicts.

---
