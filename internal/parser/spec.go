package parser

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed spec.json
var embeddedSpec []byte

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
