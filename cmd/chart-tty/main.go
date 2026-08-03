package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

const ExampleFile = "/home/codezero/workspace/github/CodeZeroSugar/chart-tty/charts/take_me_home.pro"

func main() {
	fmt.Println("Welcome to chart-tty!")

	parser := parser.NewParser(false)
	song, err := parser.ParseFile(ExampleFile)
	if err != nil {
		fmt.Printf("[Error] failed to parse file: %v", err)
		os.Exit(1)
	}

	song.Normalize()

	padding := strings.Repeat("-", 30)

	fmt.Println("TITLE: ", song.Title)
	fmt.Println("ARTIST: ", song.Artist)
	for _, section := range song.Sections {
		sectionBreak := padding + section.CleanHeader + "::" + fmt.Sprintf("%d", section.Type) + padding
		fmt.Println(sectionBreak)
		for _, row := range section.Rows {
			fmt.Println(row)
		}
	}
}
