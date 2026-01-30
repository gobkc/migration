package source

import "github.com/gobkc/migration/types"

type Source interface {
	Migrations() ([]types.Migration, error)
}
