package dialect

import (
	"strings"
	"testing"
)

func TestMySQLLockName(t *testing.T) {
	name := mysqlLockName("configurator")
	if !strings.HasPrefix(name, `gobkc_migration:`) {
		t.Fatalf("lock name %q has no prefix", name)
	}
	if len(name) > 64 {
		t.Fatalf("lock name %q exceeds MySQL's 64-character limit", name)
	}
	if name != mysqlLockName("configurator") {
		t.Fatal("lock name is not deterministic")
	}
	if name == mysqlLockName("configurator_other") {
		t.Fatal("different databases must use different lock names")
	}
}
