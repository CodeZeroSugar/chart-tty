package parser

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed spec.json
var embeddedSpec []byte

// SpecReference is the canonical ChordPro specification reference distilled
// from chordpro.org, including the chart-tty deltas section. See
// chordpro-spec.md alongside this file.
//
//go:embed chordpro-spec.md
var SpecReference string

var Spec ChordProSpec

type DirectiveCategory int

const (
	CategoryMeta DirectiveCategory = iota
	CategoryFormatting
	CategoryEnvironment
	CategoryUnknown
)

func init() {
	if err := json.Unmarshal(embeddedSpec, &Spec); err != nil {
		panic(fmt.Sprintf("failed to initialize Validator: %v\n", err))
	}
	if len(Spec.EnvironmentDirectives) == 0 || len(Spec.FormattingDirectives) == 0 || len(Spec.MetaDirectives) == 0 {
		panic("embeded spec.json contains empty directive lists\n")
	}
}

type ChordProSpec struct {
	MetaDirectives        []string `json:"meta_directives"`
	FormattingDirectives  []string `json:"formatting_directives"`
	EnvironmentDirectives []string `json:"environment_directives"`
}
