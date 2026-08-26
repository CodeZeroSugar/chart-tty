package main

import (
	"errors"
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

// runDelete removes a chart from the library by id.
func runDelete(configFlag string, id int64) {
	libPath := resolveLibraryPath(configFlag)
	store, err := db.Open(libPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: opening library: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.DeleteChart(id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "Error: chart %d not found in %s\n", id, libPath)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: deleting chart: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Deleted chart %d from %s\n", id, libPath)
}

// resolveLibraryPath returns the library database path from the config's
// [library] section, falling back to the default location.
func resolveLibraryPath(configFlag string) string {
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
	return libPath
}

// runImport imports a chart file into the library and prints the result.
func runImport(configFlag, importPath string) {
	libPath := resolveLibraryPath(configFlag)

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
	chordsFlag := flag.String("chords", "", "chord grammar: strict (default) or relaxed")
	versionFlag := flag.Bool("version", false, "print version and exit")
	aiConvertFlag := flag.Bool("ai-convert", false, "convert chart to compliant ChordPro via AI")
	writeFlag := flag.Bool("write", false, "write converted chart to <name>.pro next to source (requires --ai-convert)")
	setKeyFlag := flag.String("set-api-key", "", "store AI API key in config ('-' reads the key from stdin)")
	importFlag := flag.String("import", "", "import a chart file into the library and exit")
	deleteFlag := flag.Int("delete", 0, "delete a chart from the library by id and exit")
	initConfigFlag := flag.Bool("init-config", false, "create a default config file and exit")
	flag.Usage = usage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("chart-tty %s\n", version)
		os.Exit(0)
	}
	if *deleteFlag != 0 {
		runDelete(*configFlag, int64(*deleteFlag))
		os.Exit(0)
	}
	if *importFlag != "" {
		runImport(*configFlag, *importFlag)
		os.Exit(0)
	}
	if *initConfigFlag {
		cfgPath := *configFlag
		if cfgPath == "" {
			var err error
			cfgPath, err = config.DefaultPath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		if err := config.WriteDefault(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote default config to %s\n", cfgPath)
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

	// Resolve the chord grammar: --chords flag overrides config, default strict.
	chordModeStr := appCfg.Parser.Chords
	if *chordsFlag != "" {
		chordModeStr = *chordsFlag
	}
	chordMode, merr := parser.ParseChordMode(chordModeStr)
	if merr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", merr)
		os.Exit(2)
	}
	parser.SetDefaultChordMode(chordMode)

	rcfg := ui.RenderConfigFromConfig(appCfg)
	if *noColorFlag || os.Getenv("NO_COLOR") != "" {
		rcfg = ui.RenderConfig{}
	}
	// The TUI keeps the title and transposed key in the persistent header row,
	// so its body metadata block skips them to avoid duplication. Piped output
	// (no header) keeps the full block.
	tuiRcfg := rcfg
	tuiRcfg.SuppressMetaTitleKey = true
	isTTY := term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd()))
	aiClient := aichart.FromConfig(appCfg)

	// No chart file: launch straight into the library (interactive only).
	if len(args) == 0 {
		if *aiConvertFlag || *writeFlag || *transposeFlag != 0 {
			fmt.Fprintf(os.Stderr, "Error: --ai-convert/--write/--transpose require a chart file\n")
			os.Exit(2)
		}
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
		m := ui.NewMenuModel(store, tuiRcfg).SetKeys(appCfg.Keys).SetConverter(&aiClient)
		if err := ui.RunModel(m); err != nil {
			fmt.Fprintf(os.Stderr, "Error: TUI failed: %v\n", err)
			os.Exit(1)
		}
		return
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

	if *aiConvertFlag {
		status := ui.NewStatusLine("sending chart to " + aiClient.Model)
		res, cerr := aiClient.ConvertProgress(chart, func(e aichart.ProgressEvent) {
			status.Update(e.Message)
		})
		if cerr != nil {
			status.Finish("conversion failed")
			fmt.Fprintf(os.Stderr, "Error: AI conversion failed: %v\n", cerr)
			os.Exit(1)
		}
		status.Finish(fmt.Sprintf("converted after %d attempt(s)", res.Attempts))
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
	if (!isValid || parser.LooksLikeBasicChart(chart, chordMode)) && !*aiConvertFlag {
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

		m := ui.NewDocModel(doc, tuiRcfg).SetTranspose(*transposeFlag).SetKeys(appCfg.Keys).SetStore(store).SetSource(originalChart)
		m = m.SetConverter(&aiClient)
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
