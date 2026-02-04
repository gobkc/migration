package dbutil

import (
	"fmt"
	"testing"
)

func TestEnsureDatabase(t *testing.T) {
	dsn := `postgres://postgres:postgres@cfg-envs:5432/%s`
	dbName := "test_db"
	err := EnsureDatabase(fmt.Sprintf(dsn, dbName))
	if err != nil {
		t.Errorf("EnsureDatabase failed: %v", err)
	}
}
