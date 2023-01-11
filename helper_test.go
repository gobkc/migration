package migration

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"testing"
)

type insTest struct {
}

func (i *insTest) Version() int64 {
	return 2
}

func (i *insTest) Run() string {
	return "SELECT * FROM app limit 1"
}

func (i *insTest) Rollback() string {
	return "SELECT * FROM app_configs limit 1"
}

func (i *insTest) ChangeLog() string {
	return `only test query`
}

func TestAdd(t *testing.T) {
	type args struct {
		ins Migrates
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{
				ins: &insTest{},
			},
		},
	}
	//dsn := "postgres://postgres:123456@localhost:5432/configurator?sslmode=disable"
	dsn := ""
	if dsn == "" {
		return
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
		return
	}
	SetGorm(db)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Add(tt.args.ins)
		})
	}
	if err = AutoMigrate(); err != nil {
		t.Fatal(err)
	}
}
