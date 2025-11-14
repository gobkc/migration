package migration

import (
	"embed"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

var gdb *gorm.DB
var once sync.Once

func Setting(callbacks ...func() *gorm.DB) {
	once.Do(func() {
		gdb = &gorm.DB{}
		for _, callback := range callbacks {
			gdb = callback()
		}
	})
}

func Run(embFS embed.FS) {
	if err := acquireMigrationLock(gdb); err != nil {
		slog.Default().Error("failed to acquire migration lock", slog.String("error", err.Error()))
		return
	}
	defer releaseMigrationLock(gdb)

	if gdb == nil {
		slog.Default().Error(`Gorm DB is nil`)
		return
	}

	if err := initMigrationTable(gdb); err != nil {
		slog.Default().Error(`failed to init migration table:`, slog.String(`error`, err.Error()))
		return
	}

	lastV, err := getLastVersion(gdb)
	if err != nil {
		for {
			lastV, err = getLastVersion(gdb)
			if err != nil {
				slog.Default().Error(`failed to get last version:`, slog.String(`error`, err.Error()))
				time.Sleep(3 * time.Second)
				continue
			}
			break
		}
	}

	var version int64 = 0
	if len(lastV) > 0 {
		version = lastV[0].Version
	}

	parses := parseSql(embFS)
	tx := gdb.Begin()
	for _, pars := range parses {
		if pars.Version > version && pars.Type == TypeUp {
			if err = tx.Exec(pars.SQL.String()).Error; err != nil {
				slog.Default().Error(`failed to migrate this version:`, slog.Int64(`version`, pars.Version), slog.String(`error`, err.Error()))
				tx.Rollback()
				os.Exit(0)
			} else {
				changeLog := strings.TrimSuffix(pars.ChangeLog.String(), "\n")
				changeLog = strings.TrimPrefix(changeLog, "\n")

				if err = tx.Model(migrationsTable{}).Save(&migrationsTable{
					Version:     pars.Version,
					ChangeLog:   changeLog,
					LastMigrate: time.Now().Local(),
				}).Error; err != nil {
					tx.Rollback()
					slog.Default().Error(`failed to update migration table:`, slog.Int64(`version`, pars.Version), slog.String(`error`, err.Error()))
					os.Exit(0)
				}
				slog.Info(`successfully migrated this version:`, slog.Int64(`version`, pars.Version))
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		slog.Info(`failed to commit transaction:`, slog.String(`error`, err.Error()))
	}
}

func initMigrationTable(db *gorm.DB) error {
	if err := db.AutoMigrate(&migrationsTable{}); err != nil {
		return err
	}
	return nil
}

func getLastVersion(db *gorm.DB) (lastVersion []*migrationsTable, err error) {
	model := migrationsTable{}
	err = db.Model(model).Where(`version = (SELECT MAX(version) FROM ` + model.TableName() + `)`).Find(&lastVersion).Error
	return lastVersion, err
}

func acquireMigrationLock(db *gorm.DB) error {
	return db.Exec("SELECT pg_advisory_lock(?)", 987654321).Error
}

func releaseMigrationLock(db *gorm.DB) error {
	return db.Exec("SELECT pg_advisory_unlock(?)", 987654321).Error
}
