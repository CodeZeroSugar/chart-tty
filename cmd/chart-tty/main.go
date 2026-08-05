package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

const ExampleFile = "/home/codezero/workspace/github/CodeZeroSugar/chart-tty/charts/swing_low.pro"

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

	fmt.Println("Welcome to chart-tty!")
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
	fmt.Println("Parser Mode detected: ", parserMode)
	if parserMode == parser.ModeChordPro {
		fmt.Println("File is a valid ChordPro chart. Parser set to ChordPro mode.")
	} else {
		fmt.Println("File is not in ChordPro format. Parser set to basic mode.")
	}

	parser := parser.NewParser(parserMode)
	doc, err := parser.Parse(string(bytes))
	if err != nil {
		fmt.Println("Error: failure occurred during parsing: ", err)
	}
	pretty := doc.String()
	fmt.Println(pretty)
}
