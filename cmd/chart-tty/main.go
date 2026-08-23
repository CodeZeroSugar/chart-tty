package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
	"github.com/CodeZeroSugar/chart-tty/internal/ui"
)

func main() {
	transposeFlag := flag.Int("transpose", 0, "transpose chords by N semitones")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Please provide path to chord chart")
		os.Exit(1)
	}
	absPath, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Println("Failed to resolve path: ", args[0])
		os.Exit(1)
	}

	bytes, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Printf("Error: unable to read file: %v\n", err)
		os.Exit(1)
	}
	validator, err := parser.NewValidator()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	isValid, err := validator.ValidateChart(string(bytes))
	if !isValid {
		fmt.Printf("Error: %v", err)
	}
	parserMode := parser.DetectParserMode(isValid, err)

	parser := parser.NewParser(parserMode)
	doc, err := parser.Parse(string(bytes))
	if err != nil {
		fmt.Println("Error: failure occurred during parsing: ", err)
		os.Exit(1)
	}

	cfg := ui.RenderConfig{}
	fi, _ := os.Stdout.Stat()
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		m := ui.NewDocModel(doc, cfg).SetTranspose(*transposeFlag)
		if err := ui.RunModel(m); err != nil {
			fmt.Printf("Error: TUI failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	doc.Transpose(*transposeFlag)
	for _, line := range ui.Render(doc, cfg) {
		fmt.Println(line)
	}
}
