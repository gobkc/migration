package migration

import (
	"gorm.io/gorm"
	"sort"
	"sync"
	"time"
)

var (
	once sync.Once
	m    *migrate
)

type migItem struct {
	d       any
	version int64
}

type execItem struct {
	rollBackSql string
	changeLog   string
}

type migrate struct {
	d    []migItem
	db   *gorm.DB
	t    string // table
	exec execItem
}

func newMigrate() *migrate {
	once.Do(func() {
		m = &migrate{}
	})
	return m
}

func (m *migrate) Run(sql string, version int64) error {
	defer func() {
		m.exec = execItem{}
	}()
	if err := m.db.Exec(sql).Error; err != nil {
		m.db.Exec("UPDATE public.migrations SET version = ? WHERE id = 1", version-1)
		if rollBackErr := m.db.Exec(m.exec.rollBackSql).Error; rollBackErr != nil {
			return rollBackErr
		}
		m.db.Exec("UPDATE public.migrations SET lastRollback = ? WHERE id = 1", time.Now().Local())
		return err
	}
	err := m.db.Model(migrationsTable{}).Where("id=?", 1).Updates(migrationsTable{
		Version:   version,
		ChangeLog: m.exec.changeLog,
	}).Error
	return err
}

func (m *migrate) Rollback(sql string, version int64) {
	m.exec.rollBackSql = sql
}

func (m *migrate) ChangeLog(changeLog string, version int64) {
	m.exec.changeLog = changeLog
}

func (m *migrate) toSort() {
	sort.Slice(m.d, func(i, j int) bool {
		return m.d[i].version <= m.d[j].version
	})
}
