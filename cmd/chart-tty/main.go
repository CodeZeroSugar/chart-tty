package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/CodeZeroSugar/chart-tty/internal/aichart"
	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/db"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
	"github.com/CodeZeroSugar/chart-tty/internal/ui"
)

const version = "0.1.0"

func usage() {
	fmt.Fprintf(os.Stderr, "chart-tty %s - terminal chord chart viewer\n\n", version)
	fmt.Fprintf(os.Stderr, "Usage:\n  chart-tty [flags] <chart file>\n\nFlags:\n")
	flag.PrintDefaults()
}

// runImport imports a chart file into the library and prints the result.
func runImport(configFlag, importPath string) {
	cfgPath := configFlag
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	libPath := ""
	if appCfg, err := config.Load(cfgPath); err == nil {
		libPath = appCfg.Library.Path
	}
	if libPath == "" {
		var err error
		libPath, err = db.DefaultPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	store, err := db.Open(libPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: opening library: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	content, err := os.ReadFile(importPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to read file: %v\n", err)
		os.Exit(1)
	}

	doc, perr := parser.NewParser(parser.ModeChordPro).Parse(string(content))
	title, artist := "", ""
	if perr == nil {
		title, artist = doc.Title, doc.Artist
	}
	id, err := store.AddChart(title, artist, "import", string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: importing: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Imported %q (id %d) into %s\n", title, id, libPath)
}

func main() {
	transposeFlag := flag.Int("transpose", 0, "transpose chords by N semitones")
	configFlag := flag.String("config", "", "path to config file (default: ~/.config/chart-tty/config.toml)")
	noColorFlag := flag.Bool("no-color", false, "disable colored output")
	versionFlag := flag.Bool("version", false, "print version and exit")
	aiConvertFlag := flag.Bool("ai-convert", false, "convert chart to compliant ChordPro via AI")
	writeFlag := flag.Bool("write", false, "write converted chart to <name>.pro next to source (requires --ai-convert)")
	setKeyFlag := flag.String("set-api-key", "", "store AI API key in config ('-' reads the key from stdin)")
	importFlag := flag.String("import", "", "import a chart file into the library and exit")
	flag.Usage = usage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("chart-tty %s\n", version)
		os.Exit(0)
	}
	if *importFlag != "" {
		runImport(*configFlag, *importFlag)
		os.Exit(0)
	}
	if *writeFlag && !*aiConvertFlag {
		fmt.Fprintf(os.Stderr, "Error: --write requires --ai-convert\n")
		os.Exit(2)
	}

	cfgPath := *configFlag
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if *setKeyFlag != "" {
		key := *setKeyFlag
		if key == "-" {
			rd, rerr := io.ReadAll(os.Stdin)
			if rerr != nil {
				fmt.Fprintf(os.Stderr, "Error: reading key from stdin: %v\n", rerr)
				os.Exit(1)
			}
			key = strings.TrimSpace(string(rd))
		}
		if strings.TrimSpace(key) == "" {
			fmt.Fprintln(os.Stderr, "Error: api key must not be empty")
			os.Exit(2)
		}
		if err := config.SetAPIKey(cfgPath, key); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		masked := key
		if len(masked) > 4 {
			masked = "****" + masked[len(masked)-4:]
		}
		fmt.Printf("API key saved (%s) to %s\n", masked, cfgPath)
		os.Exit(0)
	}

	args := flag.Args()

	appCfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	rcfg := ui.RenderConfigFromConfig(appCfg)
	if *noColorFlag || os.Getenv("NO_COLOR") != "" {
		rcfg = ui.RenderConfig{}
	}
	isTTY := term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd()))

	// No chart file: launch straight into the library (interactive only).
	if len(args) == 0 {
		if !isTTY {
			flag.Usage()
			os.Exit(2)
		}
		libPath := appCfg.Library.Path
		if libPath == "" {
			libPath, err = db.DefaultPath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		store, dbErr := db.Open(libPath)
		if dbErr != nil {
			fmt.Fprintf(os.Stderr, "Error: opening library: %v\n", dbErr)
			os.Exit(1)
		}
		defer store.Close()
		m := ui.NewLibraryModel(store, rcfg).SetKeys(appCfg.Keys)
		if err := ui.RunModel(m); err != nil {
			fmt.Fprintf(os.Stderr, "Error: TUI failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *aiConvertFlag || *writeFlag || *transposeFlag != 0 {
		fmt.Fprintf(os.Stderr, "Error: --ai-convert/--write/--transpose require a chart file\n")
		os.Exit(2)
	}
	absPath, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve path %q: %v\n", args[0], err)
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
		status := newStatusLine("sending chart to " + cl.Model)
		res, cerr := cl.ConvertProgress(chart, func(e aichart.ProgressEvent) {
			status.update(e.Message)
		})
		if cerr != nil {
			status.finish("conversion failed")
			fmt.Fprintf(os.Stderr, "Error: AI conversion failed: %v\n", cerr)
			os.Exit(1)
		}
		status.finish(fmt.Sprintf("converted after %d attempt(s)", res.Attempts))
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

	if isTTY {
		libPath := appCfg.Library.Path
		if libPath == "" {
			libPath, err = db.DefaultPath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		store, dbErr := db.Open(libPath)
		if dbErr != nil {
			fmt.Fprintf(os.Stderr, "Error: opening library: %v\n", dbErr)
			os.Exit(1)
		}
		defer store.Close()

		m := ui.NewDocModel(doc, rcfg).SetTranspose(*transposeFlag).SetKeys(appCfg.Keys).SetStore(store).SetSource(originalChart)
		if converter != nil {
			m = m.SetConverter(converter)
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
