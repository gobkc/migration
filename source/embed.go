package source

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"github.com/gobkc/migration/types"
)

type Embed struct {
	FS embed.FS
}

func NewEmbed(fs embed.FS) *Embed {
	return &Embed{FS: fs}
}

func (e *Embed) Migrations() ([]types.Migration, error) {
	return parseMigrations(e.FS)
}

// 支持 final
var filePattern = regexp.MustCompile(`^(\d+)_(.+)\.(up|down|holding|final)\.sql$`)

func parseMigrations(efs embed.FS) ([]types.Migration, error) {
	var list []types.Migration

	err := fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		match := filePattern.FindStringSubmatch(d.Name())
		if match == nil {
			return nil
		}

		version, _ := strconv.ParseInt(match[1], 10, 64)
		direction := match[3]

		raw, err := efs.ReadFile(path)
		if err != nil {
			return err
		}

		sql := string(raw)

		list = append(list, types.Migration{
			Version:   version,
			Direction: direction,
			SQL:       sql,
			Checksum:  checksum(sql),
			ChangeLog: getChangelog(raw),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(list, func(i, j int) bool {
		// ⭐⭐⭐⭐⭐ note：
		// if version as the same → holding first, final last
		if list[i].Version == list[j].Version {
			return priority(list[i].Direction) < priority(list[j].Direction)
		}

		return list[i].Version < list[j].Version
	})

	return list, nil
}

func priority(dir string) int {
	switch dir {
	case types.Holding:
		return 0
	case types.Up:
		return 1
	case types.Final:
		return 2
	default:
		return 3
	}
}

func checksum(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}

func getChangelog(b []byte) string {
	var changeLog bytes.Buffer
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	lines := bytes.SplitSeq(b, []byte("\n"))

	for line := range lines {
		newLine := bytes.TrimSpace(line)

		if len(newLine) == 0 {
			continue
		}

		if bytes.HasPrefix(newLine, []byte("//")) || bytes.HasPrefix(newLine, []byte("--")) {
			changeLog.Write(bytes.TrimSpace(line[2:]))
			changeLog.Write([]byte{10})
		}
	}

	return changeLog.String()
}
