// builtin format-independent macros

package frundis

import (
	"bytes"
	"math"
	"strings"

	"github.com/anaseto/gofrundis/ast"
)

func macroDefStart(exp BaseExporter) {
	// macro .#de
	bctx := exp.BaseContext()
	if bctx.defInfo != nil {
		bctx.Error("not allowed in the scope of a previous `.#de' at line ",
			bctx.defInfo.line, " of file ", bctx.defInfo.file)
		return
	}
	opts, _, args := bctx.ParseOptions(specOptIf, bctx.Args)
	if len(args) < 1 {
		bctx.Error("'.#de' requires name argument")
		return
	}
	ignore := false
	if fmt, okFmt := opts["f"]; okFmt {
		formats := strings.Split(bctx.InlinesToText(fmt), ",")
		bctx.checkFormats(formats)
		if bctx.notExportFormat(formats) {
			ignore = true
		}
	}
	name := bctx.InlinesToText(args[0])
	bctx.defInfo = &macroDefInfo{
		name:   name,
		blocks: []ast.Block{},
		line:   bctx.line,
		ignore: ignore,
		file:   bctx.loc.curFile}
}

func macroDefEnd(exp BaseExporter) {
	// macro .#.
	bctx := exp.BaseContext()
	if bctx.defInfo == nil {
		bctx.Error("found '.#.' without previous '.#de'")
		return
	}
	if !bctx.defInfo.ignore {
		bctx.defInfo.argsc, bctx.defInfo.opts = bctx.searchArgInBlocks(bctx.defInfo.blocks)
		bctx.macros[bctx.defInfo.name] = *bctx.defInfo
	}
	bctx.defInfo = nil
}

// searchArgInBlocks returns the greatest number N of an $N argument, as well
// as a specification of macro options found.
func (bctx *BaseContext) searchArgInBlocks(blocks []ast.Block) (int, map[string]Option) {
	max := 0
	opts := make(map[string]Option)
	for _, b := range blocks {
		m := bctx.searchArgInBlock(b, opts)
		if m > max {
			max = m
		}
	}
	return max, opts
}

func (bctx *BaseContext) searchArgInBlock(b ast.Block, opts map[string]Option) int {
	max := 0
	switch b := b.(type) {
	case *ast.Macro:
		for _, arg := range b.Args {
			m := bctx.searchArgInText(arg, opts)
			if m > max {
				max = m
			}
		}
	case *ast.TextBlock:
		m := bctx.searchArgInText(b.Text, opts)
		if m > max {
			max = m
		}
	}
	return max
}

func (bctx *BaseContext) searchArgInText(text []ast.Inline, opts map[string]Option) int {
	max := 0
	for _, elt := range text {
		switch elt := elt.(type) {
		case ast.ArgEscape:
			if int(elt) > max {
				max = int(elt)
			}
		case ast.NamedArgEscape:
			opt, ok := opts[string(elt)]
			if !ok {
				opts[string(elt)] = ArgOption
			} else if opt != ArgOption {
				bctx.Error("both as flag and option with argument:", elt)
			}
		case ast.NamedFlagEscape:
			opt, ok := opts[string(elt)]
			if !ok {
				opts[string(elt)] = FlagOption
			} else if opt != FlagOption {
				bctx.Error("both as flag and option with argument:", elt)
			}
		case ast.Escape:
			if elt == ast.Escape("$@") {
				return math.MaxInt32
			}
		}
	}
	return max
}

func macroIfStart(exp BaseExporter) {
	// macro .#if
	bctx := exp.BaseContext()
	bctx.pushScope(&scope{name: "#if"})
	if bctx.ifIgnore > 0 {
		bctx.ifIgnore++
		return
	}
	opts, flags, args := bctx.ParseOptions(specOptIf, bctx.Args)
	fmt, ok := opts["f"]
	if !(len(args) > 0) && !ok {
		bctx.Error("useless `.#if' invocation")
	}
	if ok {
		formats := strings.Split(bctx.InlinesToText(fmt), ",")
		bctx.checkFormats(formats)
		if bctx.notExportFormat(formats) {
			bctx.ifIgnore = 1
			return
		}
	}
	if len(args) > 0 {
		if len(args) > 1 {
			bctx.Error("too many arguments")
		}
		var flag int
		switch bctx.InlinesToText(args[0]) {
		case "0", "":
			flag = 0
		default:
			flag = 1
		}
		if flags["not"] {
			flag = 1 - flag
		}
		bctx.ifIgnore = 1 - flag
	}
}

func macroIfEnd(exp BaseExporter) {
	// macro .#;
	bctx := exp.BaseContext()
	if bctx.ifIgnore > 0 {
		bctx.ifIgnore--
	}
	scope := bctx.popScope("#if")
	if scope == nil {
		bctx.Error("no corresponding `.#if'")
	}
}

func macroDefVar(exp BaseExporter) {
	// macro .#dv
	bctx := exp.BaseContext()
	opts, _, args := bctx.ParseOptions(specOptDefVar, bctx.Args)
	if len(args) == 0 {
		bctx.Error("requires a name argument")
		return
	}
	if fmt, ok := opts["f"]; ok {
		formats := strings.Split(bctx.InlinesToText(fmt), ",")
		bctx.checkFormats(formats)
		if bctx.notExportFormat(formats) {
			return
		}
	}
	name := bctx.InlinesToText(args[0])
	args = args[1:]
	buf := bytes.Buffer{}
	for i, arg := range args {
		if i > 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(bctx.InlinesToText(arg))
	}
	bctx.vars[name] = buf.String()
}

func macroSource(exp BaseExporter) {
	// macro .#so
	bctx := exp.BaseContext()
	args := bctx.Args
	if len(args) < 1 {
		bctx.Error("filename argument required")
		return
	}
	filename := bctx.InlinesToText(args[0])
	err := processFile(exp, filename)
	if err != nil {
		bctx.Error(err)
	}
}
