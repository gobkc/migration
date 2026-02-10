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

var filePattern = regexp.MustCompile(`^(\d+)_(.+)\.(up|down|holding)\.sql$`)

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

		changeLog := getChangelog(raw)

		sql := string(raw)

		list = append(list, types.Migration{
			Version:   version,
			Direction: direction,
			SQL:       sql,
			Checksum:  checksum(sql),
			ChangeLog: changeLog,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Version < list[j].Version
	})

	return list, nil
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
