package migration

import (
	"embed"
	"testing"
)

//go:embed migrations/*
var Files embed.FS

func Test_parseSql(t *testing.T) {
	parseSql(Files)
}
