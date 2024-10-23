package migration

import "time"

const (
	TypeUp   = `up`
	TypeDown = `down`
)

type migrationsTable struct {
	Id           int64      `json:"id" gorm:"primaryKey"`
	Version      int64      `json:"version"`
	ChangeLog    string     `json:"change_log"`
	LastMigrate  time.Time  `gorm:"autoUpdateTime" json:"last_migrate"`
	LastRollback *time.Time `json:"last_rollback"`
}

func (migrationsTable) TableName() string {
	return "migrations"
}
