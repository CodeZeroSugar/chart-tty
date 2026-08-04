package main

import (
	"fmt"
	"os"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

const ExampleFile = "/home/codezero/workspace/github/CodeZeroSugar/chart-tty/charts/boulevard.pro"

func main() {
	fmt.Println("Welcome to chart-tty!")
	bytes, err := os.ReadFile(ExampleFile)
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
	_, err = parser.Parse(string(bytes))
	if err != nil {
		fmt.Println("Error: failure occurred during parsing: ", err)
	}
}
