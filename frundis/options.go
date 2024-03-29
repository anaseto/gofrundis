// Macro option parsing

package frundis

import "codeberg.org/anaseto/gofrundis/ast"

// Option represents the type of option (flag or with argument).
type Option int

// Option types
const (
	FlagOption Option = iota // boolean flag
	ArgOption                // option with argument
)

// ParseOptions takes a specification spec of options and a arguments slice
// args and returns a mapping from option names to values, a mapping for flags
// and an updated arguments slice.
func (ctx *Context) ParseOptions(
	spec map[string]Option, args [][]ast.Inline) (
	map[string][]ast.Inline, map[string]bool, [][]ast.Inline) {

	var opts map[string][]ast.Inline
	var flags map[string]bool
scanOptions:
	for len(args) > 0 {
		flag := args[0]
		if len(flag) == 0 {
			break scanOptions
		}
		switch b := flag[0].(type) {
		case ast.Text:
			if len(b) == 0 || b[0] != '-' {
				break scanOptions
			}
		default:
			break scanOptions
		}
		name := ctx.InlinesToText(flag)[1:]
		args = args[1:]
		optionType, ok := spec[name]
		if !ok {
			ctx.Errorf("unrecognized option: -%s", name)
			continue scanOptions
		}
		if optionType == ArgOption {
			if len(args) == 0 {
				ctx.Errorf("option -%s requires an argument", name)
				continue scanOptions
			}
			arg := args[0]
			args = args[1:]
			if !(len(arg) > 0 && arg[0] == ast.Text("-")) {
				if opts == nil {
					opts = make(map[string][]ast.Inline)
				}
				opts[name] = arg
			}
		} else {
			if flags == nil {
				flags = make(map[string]bool)
			}
			flags[name] = true
		}
	}
	return opts, flags, args
}

var specOptBd = map[string]Option{
	"t":  ArgOption,
	"r":  FlagOption,
	"id": ArgOption}
var specOptBf = map[string]Option{
	"t":  ArgOption,
	"f":  ArgOption,
	"ns": FlagOption}
var specOptBl = map[string]Option{
	"id":      ArgOption,
	"t":       ArgOption,
	"columns": ArgOption}
var specOptBm = map[string]Option{
	"t":  ArgOption,
	"r":  FlagOption,
	"ns": FlagOption,
	"id": ArgOption}
var specOptD = map[string]Option{}
var specOptDef = map[string]Option{"f": ArgOption}
var specOptDefVar = map[string]Option{"f": ArgOption}
var specOptEd = map[string]Option{"t": ArgOption}
var specOptEl = map[string]Option{}
var specOptEm = map[string]Option{
	"t":  ArgOption,
	"ns": FlagOption,
}
var specOptEf = map[string]Option{"ns": FlagOption}
var specOptFt = map[string]Option{
	"t":  ArgOption,
	"f":  ArgOption,
	"ns": FlagOption}
var specOptIf = map[string]Option{
	"eq":  ArgOption,
	"f":   ArgOption,
	"not": FlagOption}
var specOptIncludeFile = map[string]Option{
	"f":     ArgOption,
	"ns":    FlagOption,
	"as-is": FlagOption,
	"t":     ArgOption}
var specOptIm = map[string]Option{
	"alt":  ArgOption,
	"id":   ArgOption,
	"ns":   FlagOption,
	"link": ArgOption}
var specOptIt = map[string]Option{}
var specOptLk = map[string]Option{"ns": FlagOption}
var specOptP = map[string]Option{}
var specOptRun = map[string]Option{}
var specOptSm = map[string]Option{
	"t":  ArgOption,
	"ns": FlagOption,
	"id": ArgOption}
var specOptSx = map[string]Option{
	"ns": FlagOption}
var specOptTa = map[string]Option{}
var specOptTc = map[string]Option{
	"summary": FlagOption,
	"nonum":   FlagOption,
	"mini":    FlagOption,
	"toc":     FlagOption,
	"lof":     FlagOption,
	"lot":     FlagOption,
	"lop":     FlagOption,
	"title":   ArgOption}
var specOptXdtag = map[string]Option{
	"t": ArgOption,
	"f": ArgOption,
	"a": ArgOption,
	"c": ArgOption}
var specOptXftag = map[string]Option{
	"t":      ArgOption,
	"f":      ArgOption,
	"shell":  FlagOption,
	"gsub":   ArgOption,
	"regexp": ArgOption}
var specOptXmtag = map[string]Option{
	"t": ArgOption,
	"f": ArgOption,
	"c": ArgOption,
	"a": ArgOption,
	"b": ArgOption,
	"e": ArgOption}
var specOptXset = map[string]Option{"f": ArgOption}
var specOptHeader = map[string]Option{
	"id":    ArgOption,
	"nonum": FlagOption}
