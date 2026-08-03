package main

import (
	"fmt"
	"os"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

const ExampleFile = "/home/codezero/workspace/github/CodeZeroSugar/chart-tty/charts/take_me_home.pro"

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
		os.Exit(1)
	}

	fmt.Println("File is a valid ChordPro chart!")
}
