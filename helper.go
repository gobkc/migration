package migration

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
)

// Add note
// "ins" is a structure instance that implements migrates
// No need to implement all methods
func Add(ins Migrates) {
	mig := newMigrate()
	if mig.db == nil {
		panic("not set Gorm connection")
	}
	version := getVersion()
	mig.d = append(mig.d, migItem{
		d:       ins,
		version: version,
	})
}

func SetGorm(db *gorm.DB) {
	mig := newMigrate()
	mig.db = db
}

// AutoMigrate It needs to be written after the "Add" method and finally executed to ensure correct migration
func AutoMigrate() error {
	mig := newMigrate()
	findVersion := findOrInitVersion()
	version := findVersion.Version
	mig.toSort() //sort mig.d
	for _, item := range mig.d {
		if item.version <= version {
			continue
		}
		if err := checkStruct(item.d); err != nil {
			log.Println(err.Error())
			continue
		}
		itemType := reflect.TypeOf(item.d)
		itemValue := reflect.ValueOf(item.d)
		methodNums := itemValue.NumMethod()
		for i := 0; i < methodNums; i++ {
			methodName := itemType.Method(i).Name
			result := itemValue.MethodByName(methodName).Call([]reflect.Value{})
			if len(result) == 0 {
				continue
			}
			param := []reflect.Value{reflect.ValueOf(result[0].String()), reflect.ValueOf(item.version)}
			// call migration same method
			exec := reflect.ValueOf(m).MethodByName(methodName).Call(param)
			if len(exec) == 0 {
				continue
			}
			if execErr := exec[0].Interface(); execErr != nil {
				return fmt.Errorf("an error occurred while migrating database version %v:%v", item.version, execErr)
			}
		}
	}
	return nil
}

func findOrInitVersion() *migrationsTable {
	mig := newMigrate()
	if findMigTable := mig.db.Migrator().HasTable(&migrationsTable{}); !findMigTable {
		mig.db.Migrator().CreateTable(&migrationsTable{})
	}
	migTable := &migrationsTable{}
	err := mig.db.Model(migrationsTable{}).Attrs(migrationsTable{Id: 1}).FirstOrCreate(&migTable).Error
	if err != nil {
		log.Println(err)
	}
	return migTable
}

func getVersion() int64 {
	_, file, _, _ := runtime.Caller(1)
	parse := regexp.MustCompile(`/([0-9]*?).go`)
	parseArr := parse.FindAllStringSubmatch(file, -1)
	fmt.Println(parseArr)
	var version int64 = 0
	if len(parseArr) > 0 {
		if len(parseArr[0]) > 1 {
			version, _ = strconv.ParseInt(parseArr[0][1], 10, 64)
		}
	}
	return version
}

func checkStruct(dest any) error {
	switch kind := reflect.ValueOf(dest).Kind(); kind {
	case reflect.Pointer:
		if reKind := reflect.ValueOf(dest).Elem().Kind(); reKind != reflect.Struct {
			return errors.New("dest must be a pointer structure")
		}
	default:
		return errors.New("dest must be a pointer structure")
	}
	return nil
}
