package migration

import (
	"bytes"
	"embed"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"unicode"
)

type ParseItem struct {
	Version   int64
	ChangeLog bytes.Buffer
	SQL       bytes.Buffer
	Type      string
}

func parseSql(embFS embed.FS) []*ParseItem {
	var parses []*ParseItem
	fs.WalkDir(embFS, `.`, func(path string, dir fs.DirEntry, err error) error {
		if !dir.IsDir() {
			fileName := dir.Name()
			var sType = TypeDown
			if strings.Contains(fileName, `.`+TypeUp+`.`) {
				sType = TypeUp
			}
			if strings.Contains(fileName, `.`+TypeDown+`.`) {
				sType = TypeDown
			}
			version := fileNameConvertToVersion(fileName)
			content, err := embFS.ReadFile(path)
			if err != nil {
				slog.Default().Error(`Failed to read file`, slog.String(`error`, err.Error()), slog.String(`file`, fileName))
				return err
			}
			item := fileContentConvertToParseItem(content)
			item.Type = sType
			item.Version = version
			parses = append(parses, item)
		}
		return nil
	})
	sort.Slice(parses, func(i, j int) bool {
		return parses[i].Version <= parses[j].Version
	})

	return parses
}

func fileNameConvertToVersion(fileName string) int64 {
	var version bytes.Buffer
	for _, r := range fileName {
		if unicode.IsDigit(r) {
			version.WriteRune(r)
		} else {
			break
		}
	}
	var v int64 = 0
	for _, b := range version.Bytes() {
		if b < '0' || b > '9' {
			continue
		}
		v = v*10 + int64(byte(b)-'0')
	}
	return v
}

func fileContentConvertToParseItem(b []byte) *ParseItem {
	item := &ParseItem{}
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	lines := bytes.Split(b, []byte("\n"))
	for _, line := range lines {
		newLine := bytes.TrimSpace(line)
		if len(newLine) == 0 {
			continue
		}
		if bytes.HasPrefix(newLine, []byte("//")) {
			item.ChangeLog.Write(bytes.TrimSpace(line[2:]))
			item.ChangeLog.Write([]byte{10})
		} else {
			item.SQL.Write(line)
			item.SQL.Write([]byte{10})
		}
	}
	return item
}
