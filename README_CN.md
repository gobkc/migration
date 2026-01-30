好的，我为你的工业级 Go Migration 库写一个完整的 `README.md`，风格清晰、面向开发者使用，包含项目介绍、使用方法和注意事项：

---

# README.md

```markdown
# Go Migration Library

一个轻量级、工业级的 Golang 数据库迁移库，支持 **PostgreSQL** 和 **MySQL**。  
专为多 Pod、云原生环境设计，**无需额外 lock 表**，支持跨数据库锁、dirty 防护、checksum 校验和 execution time 记录。

---

## 特性

- ✅ 支持 PostgreSQL 与 MySQL
- ✅ 跨数据库安全锁（无需 lock 表）
- ✅ dirty 状态防止中断迁移造成数据库不一致
- ✅ 文件 checksum 校验，防止已执行 SQL 被修改
- ✅ 支持 embed.FS 或自定义 Source
- ✅ 执行时间记录每个 migration
- ✅ Up / Down 迁移模式
- ✅ 易于扩展 Dialect（支持多数据库 placeholder、类型等）
- ✅ 适合多 Pod / 多实例部署

---

## 文件格式规范

迁移文件必须符合命名规则：

```

<version>*<description>.up.sql <version>*<description>.down.sql

```

例如：

```

202502021200_create_user_table.up.sql
202502021200_create_user_table.down.sql

````

- `<version>`：整数或时间戳，唯一标识迁移版本  
- `<description>`：描述，可读字符串  
- `.up.sql`：向前迁移  
- `.down.sql`：回滚迁移  

---

## 安装

```bash
go get github.com/gobkc/migration
````

---

## 使用方法

### 1. embed FS 示例

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

    _ "github.com/lib/pq" // postgres driver
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

### 2. 主要 API

```go
type Migrator struct {
    db      *sql.DB
    dialect dialect.Dialect
    source  source.Source
}

// 创建 Migrator
func New(db *sql.DB, d dialect.Dialect, s source.Source) *Migrator

// 执行 Up Migration
func (m *Migrator) Up(ctx context.Context) error
```

* `db`：数据库连接
* `dialect`：数据库方言（Postgres/MySQL）
* `source`：迁移文件来源（Embed / 自定义）

---

## 注意事项

1. **dirty 状态保护**

   * 如果上次迁移未完成，`migrations` 中 dirty=true，迁移将被拒绝
   * 需要手动修复或使用自定义 repair 方法

2. **checksum 校验**

   * 已执行 SQL 的 checksum 会被记录
   * 如果文件被修改，迁移会拒绝执行，防止 schema 不一致

3. **锁机制**

   * 使用 `version = -1` 的 leader row 实现跨 Pod 安全锁
   * 只有一个实例可以执行迁移
   * 失败或冲突时会返回 `ErrLocked`

4. **多 Pod 部署建议**

   * 推荐在 Kubernetes 中使用 initContainer 执行 migration
   * 业务应用容器启动时不应自动运行迁移

5. **Down Migration**

   * 支持回滚，但务必确保回滚 SQL 的正确性
   * Up / Down 文件必须一一对应

6. **Dialect 扩展**

   * 如需支持其他数据库，可实现 `Dialect` 接口

---

## 表结构（跨数据库通用）

```sql
CREATE TABLE migrations (
    version BIGINT PRIMARY KEY,
    checksum VARCHAR(64) NOT NULL,
    applied_at TIMESTAMP NOT NULL,
    execution_time_s BIGINT NOT NULL,
    dirty BOOLEAN NOT NULL
);
```

* `version`：迁移版本
* `checksum`：SQL 文件 SHA256 校验码
* `applied_at`：执行时间
* `execution_time_s`：执行耗时
* `dirty`：迁移未完成标记

> `version = -1` 用作锁

---

## 推荐使用方式

* 将 migration 执行放在 **单独命令或 initContainer**
* 确保多 Pod 部署时，只有一个实例运行 migration
* 不要在应用启动时直接调用 Up，避免竞争

---
