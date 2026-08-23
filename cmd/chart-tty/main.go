package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
	"github.com/CodeZeroSugar/chart-tty/internal/ui"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Please provide path to chord chart")
		os.Exit(1)
	}
	absPath, err := filepath.Abs(args[1])
	if err != nil {
		fmt.Println("Failed to resolve path: ", args[1])
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
		if err := ui.Run(doc, cfg); err != nil {
			fmt.Printf("Error: TUI failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	for _, line := range ui.Render(doc, cfg) {
		fmt.Println(line)
	}
}
