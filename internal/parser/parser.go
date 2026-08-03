// Package parser internal/parser/parser.go
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Parser struct {
	IncludeTabs bool
}

func NewParser(includeTabs bool) *Parser {
	return &Parser{IncludeTabs: includeTabs}
}

func (p *Parser) ParseFile(filePath string) (*Song, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return &Song{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	song := &Song{}
	var section SongSection

	isTabBlock := false
	var tabBuf []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "{t:") || strings.HasPrefix(line, "{title:") {
			song.Title = strings.TrimSuffix(strings.Split(line, ":")[1], "}")
			continue
		}
		if strings.HasPrefix(line, "{st:") || strings.HasPrefix(line, "{subtitle:") {
			song.Artist = strings.TrimSuffix(strings.Split(line, ":")[1], "}")
			continue
		}

		if strings.HasPrefix(line, "{sot}") {
			isTabBlock = true
			tabBuf = []string{line}
			continue
		}

		if isTabBlock {
			tabBuf = append(tabBuf, line)
			if line == "{eot}" {
				isTabBlock = false
				if isValidTab(tabBuf) {
					section.Rows = append(section.Rows, tabBuf...)
				}
			}
			continue
		}

		if line == "" {
			if len(section.Rows) > 0 {
				section.RawHeader = section.Rows[0]
				song.Sections = append(song.Sections, section)
				section = SongSection{}

			}
			continue
		}
		section.Rows = append(section.Rows, line)

	}

	if len(section.Rows) > 0 {
		section.RawHeader = section.Rows[0]
		song.Sections = append(song.Sections, section)
	}

	return song, scanner.Err()
}

func isValidTab(tabBlock []string) bool {
	return len(tabBlock) > 3
}
