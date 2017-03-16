// builtin format-independent macros

package frundis

import (
	"bytes"
	"math"
	"os"
	"strings"

	"github.com/anaseto/gofrundis/ast"
)

func macroDefStart(exp Exporter) {
	// macro .#de
	ctx := exp.Context()
	if ctx.uMacroDef != nil {
		ctx.Error("not allowed in the scope of a previous `.#de' at line ",
			ctx.uMacroDef.line, " of file ", ctx.uMacroDef.file)
		return
	}
	opts, _, args := ctx.ParseOptions(specOptIf, ctx.Args)
	if len(args) < 1 {
		ctx.Error("'.#de' requires name argument")
		return
	}
	ignore := false
	if fmt, okFmt := opts["f"]; okFmt {
		formats := strings.Split(ctx.InlinesToText(fmt), ",")
		ctx.checkFormats(formats)
		if ctx.notExportFormat(formats) {
			ignore = true
		}
	}
	name := ctx.InlinesToText(args[0])
	ctx.uMacroDef = &uMacroDefInfo{
		name:   name,
		blocks: []ast.Block{},
		line:   ctx.line,
		ignore: ignore,
		file:   ctx.loc.curFile}
}

func macroDefEnd(exp Exporter) {
	// macro .#.
	ctx := exp.Context()
	if len(ctx.Args) > 0 {
		ctx.Error("useless arguments")
	}
	if ctx.uMacroDef == nil {
		ctx.Error("found '.#.' without previous '.#de'")
		return
	}
	if !ctx.uMacroDef.ignore {
		ctx.uMacroDef.argsc, ctx.uMacroDef.opts = ctx.searchArgInBlocks(ctx.uMacroDef.blocks)
		ctx.uMacros[ctx.uMacroDef.name] = *ctx.uMacroDef
	}
	ctx.uMacroDef = nil
}

// searchArgInBlocks returns the greatest number N of an $N argument, as well
// as a specification of macro options found.
func (ctx *Context) searchArgInBlocks(blocks []ast.Block) (int, map[string]Option) {
	max := 0
	opts := make(map[string]Option)
	for _, b := range blocks {
		m := ctx.searchArgInBlock(b, opts)
		if m > max {
			max = m
		}
	}
	return max, opts
}

func (ctx *Context) searchArgInBlock(b ast.Block, opts map[string]Option) int {
	max := 0
	switch b := b.(type) {
	case *ast.Macro:
		for _, arg := range b.Args {
			m := ctx.searchArgInText(arg, opts)
			if m > max {
				max = m
			}
		}
	case *ast.TextBlock:
		m := ctx.searchArgInText(b.Text, opts)
		if m > max {
			max = m
		}
	}
	return max
}

func (ctx *Context) searchArgInText(text []ast.Inline, opts map[string]Option) int {
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
				ctx.Error("both as flag and option with argument:", elt)
			}
		case ast.NamedFlagEscape:
			opt, ok := opts[string(elt)]
			if !ok {
				opts[string(elt)] = FlagOption
			} else if opt != FlagOption {
				ctx.Error("both as flag and option with argument:", elt)
			}
		case ast.Escape:
			if elt == ast.Escape("$@") {
				max = math.MaxInt32
			}
		}
	}
	return max
}

func macroIfStart(exp Exporter) {
	// macro .#if
	ctx := exp.Context()
	ctx.pushScope(&scope{name: "#if"})
	if ctx.ifIgnoreDepth > 0 {
		ctx.ifIgnoreDepth++
		return
	}
	opts, flags, args := ctx.ParseOptions(specOptIf, ctx.Args)
	fmt, ok := opts["f"]
	if !(len(args) > 0) && !ok {
		ctx.Error("useless `.#if' invocation")
	}
	if ok {
		formats := strings.Split(ctx.InlinesToText(fmt), ",")
		ctx.checkFormats(formats)
		if ctx.notExportFormat(formats) {
			ctx.ifIgnoreDepth = 1
			return
		}
	}
	if len(args) > 0 {
		if len(args) > 1 {
			ctx.Error("too many arguments")
		}
		var flag int
		switch ctx.InlinesToText(args[0]) {
		case "0", "":
			flag = 0
		default:
			flag = 1
		}
		if flags["not"] {
			flag = 1 - flag
		}
		ctx.ifIgnoreDepth = 1 - flag
	}
}

func macroIfEnd(exp Exporter) {
	// macro .#;
	ctx := exp.Context()
	if len(ctx.Args) > 0 {
		ctx.Error("useless arguments")
	}
	if ctx.ifIgnoreDepth > 0 {
		ctx.ifIgnoreDepth--
	}
	scope := ctx.popScope("#if")
	if scope == nil {
		ctx.Error("no corresponding `.#if'")
	}
}

func macroDefVar(exp Exporter) {
	// macro .#dv
	ctx := exp.Context()
	opts, _, args := ctx.ParseOptions(specOptDefVar, ctx.Args)
	if len(args) == 0 {
		ctx.Error("requires a name argument")
		return
	}
	if fmt, ok := opts["f"]; ok {
		formats := strings.Split(ctx.InlinesToText(fmt), ",")
		ctx.checkFormats(formats)
		if ctx.notExportFormat(formats) {
			return
		}
	}
	name := ctx.InlinesToText(args[0])
	args = args[1:]
	buf := bytes.Buffer{}
	for i, arg := range args {
		if i > 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(ctx.InlinesToText(arg))
	}
	ctx.ivars[name] = buf.String()
}

func macroRun(exp Exporter) {
	// macro .#run
	ctx := exp.Context()
	if !ctx.Unrestricted {
		ctx.Error("skipping disallowed external command")
		return
	}
	_, _, args := ctx.ParseOptions(specOptRun, ctx.Args)
	if !ctx.Process {
		// NOTE: it could eventually be interesting to add an option
		// that populates stdin of command with information which the
		// command could use to customize behavior, and even to collect
		// data during info pass.
		return
	}
	sargs := make([]string, 0, len(args))
	for _, elt := range args {
		sargs = append(sargs, ctx.InlinesToText(elt))
	}
	if len(sargs) < 1 {
		ctx.Error("not enough arguments")
		return
	}
	cmd := getCommand(sargs)
	cmd.Stderr = os.Stderr
	bytes, err := cmd.Output()
	if err != nil {
		ctx.Error("shell command:", sargs, ":", err)
		return
	}
	ctx.W().Write(bytes)
}
