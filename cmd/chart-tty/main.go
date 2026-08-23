package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/CodeZeroSugar/chart-tty/internal/aichart"
	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
	"github.com/CodeZeroSugar/chart-tty/internal/ui"
)

const version = "0.1.0"

func usage() {
	fmt.Fprintf(os.Stderr, "chart-tty %s - terminal chord chart viewer\n\n", version)
	fmt.Fprintf(os.Stderr, "Usage:\n  chart-tty [flags] <chart file>\n\nFlags:\n")
	flag.PrintDefaults()
}

func main() {
	transposeFlag := flag.Int("transpose", 0, "transpose chords by N semitones")
	configFlag := flag.String("config", "", "path to config file (default: ~/.config/chart-tty/config.toml)")
	noColorFlag := flag.Bool("no-color", false, "disable colored output")
	versionFlag := flag.Bool("version", false, "print version and exit")
	aiConvertFlag := flag.Bool("ai-convert", false, "convert chart to compliant ChordPro via AI")
	writeFlag := flag.Bool("write", false, "write converted chart to <name>.pro next to source (requires --ai-convert)")
	flag.Usage = usage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("chart-tty %s\n", version)
		os.Exit(0)
	}
	if *writeFlag && !*aiConvertFlag {
		fmt.Fprintf(os.Stderr, "Error: --write requires --ai-convert\n")
		os.Exit(2)
	}
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(2)
	}
	absPath, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve path %q: %v\n", args[0], err)
		os.Exit(1)
	}

	cfgPath := *configFlag
	if cfgPath == "" {
		cfgPath, err = config.DefaultPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	appCfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	bytes, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to read file: %v\n", err)
		os.Exit(1)
	}
	chart := string(bytes)
	originalChart := chart

	var converter *aichart.Client
	if *aiConvertFlag {
		cl := aichart.FromConfig(appCfg)
		converter = &cl
		res, cerr := cl.Convert(chart)
		if cerr != nil {
			fmt.Fprintf(os.Stderr, "Error: AI conversion failed: %v\n", cerr)
			os.Exit(1)
		}
		if *writeFlag {
			ext := filepath.Ext(absPath)
			outPath := strings.TrimSuffix(absPath, ext) + ".pro"
			if err := os.WriteFile(outPath, []byte(res.Chart), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: writing converted chart: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Wrote converted chart to %s\n", outPath)
		}
		chart = res.Chart
	}

	validator, err := parser.NewValidator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	isValid, verr := validator.ValidateChart(chart)

	mode := parser.ModeChordPro
	if (!isValid || parser.LooksLikeBasicChart(chart)) && !*aiConvertFlag {
		if !isValid {
			fmt.Fprintf(os.Stderr, "Warning: chart is not valid ChordPro (%v); falling back to basic mode\n", verr)
		}
		mode = parser.ModeBasic
	}

	doc, err := parser.NewParser(mode).Parse(chart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: parsing failed: %v\n", err)
		os.Exit(1)
	}

	rcfg := ui.RenderConfigFromConfig(appCfg)
	if *noColorFlag || os.Getenv("NO_COLOR") != "" {
		rcfg = ui.RenderConfig{}
	}

	isTTY := term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd()))
	if isTTY {
		m := ui.NewDocModel(doc, rcfg).SetTranspose(*transposeFlag).SetKeys(appCfg.Keys)
		if converter != nil {
			m = m.SetConverter(originalChart, converter)
		}
		if err := ui.RunModel(m); err != nil {
			fmt.Fprintf(os.Stderr, "Error: TUI failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	doc.Transpose(*transposeFlag)
	for _, line := range ui.Render(doc, rcfg) {
		fmt.Println(line)
	}
}