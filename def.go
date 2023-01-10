package migration

import "time"

type Migrates interface {
	Run() string
	Rollback() string
	ChangeLog() string
}

type migrationsTable struct {
	Id           int64     `json:"id" gorm:"primaryKey"`
	Version      int64     `json:"version"`
	ChangeLog    string    `json:"changeLog"`
	LastMigrate  time.Time `gorm:"autoUpdateTime" json:"lastMigrate"`
	LastRollback time.Time `json:"lastRollback"`
}

func (migrationsTable) TableName() string {
	return "migrations"
}
