package types

type Migration struct {
	Version   int64
	Direction string
	SQL       string
	Checksum  string
	ChangeLog string
}

const BaseLineVersion int64 = 0
