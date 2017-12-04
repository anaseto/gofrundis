// User macro processing stuff

package frundis

import (
	"fmt"

	"github.com/anaseto/gofrundis/ast"
)

// argsSubst returns inline text where numbered arguments are substituted with
// values from args
func (ctx *Context) argsSubstText(
	m *uMacroDefInfo,
	args [][]ast.Inline, opts map[string][]ast.Inline,
	flags map[string]bool, text []ast.Inline) []ast.Inline {

	res := []ast.Inline{}
scanText:
	for _, elt := range text {
		switch elt := elt.(type) {
		case ast.ArgEscape:
			if int(elt) > len(args) || int(elt) <= 0 {
				if ctx.Process {
					ctx.Errorf("missing argument: $%v", elt)
				}
				res = append(res, ast.Text(fmt.Sprint("\\$", int(elt))))
				continue scanText
			}
			res = append(res, args[elt-1]...)
		case ast.NamedArgEscape:
			arg, ok := opts[string(elt)]
			if !ok {
				if ctx.Process {
					ctx.Errorf("missing named argument: $[%s]", string(elt))
				}
				continue scanText
			}
			res = append(res, arg...)
		case ast.NamedFlagEscape:
			flag := flags[string(elt)]
			var boolText string
			if flag {
				boolText = "1"
			}
			res = append(res, ast.Text(boolText))
		case ast.Escape:
			if elt == ast.Escape("$@") {
				min := len(args)
				if min > m.argsc {
					min = m.argsc
				}
				for i, arg := range args[min:] {
					if i > 0 {
						res = append(res, ast.Text(" "))
					}
					res = append(res, arg...)
				}
			} else {
				res = append(res, elt)
			}
		default:
			res = append(res, elt)
		}
	}
	return res
}

func (ctx *Context) argsSubstBlock(
	m *uMacroDefInfo,
	args [][]ast.Inline, opts map[string][]ast.Inline,
	flags map[string]bool, b ast.Block) ast.Block {

	var res ast.Block
	switch b := b.(type) {
	case *ast.Macro:
		nargs := [][]ast.Inline{}
		for _, arg := range b.Args {
			if len(arg) == 1 && arg[0] == ast.Escape("$@") {
				min := len(args)
				if min > m.argsc {
					min = m.argsc
				}
				nargs = append(nargs, args[min:]...)
				continue
			}
			narg := ctx.argsSubstText(m, args, opts, flags, arg)
			nargs = append(nargs, narg)
		}
		res = &ast.Macro{
			Args: nargs,
			Line: b.Line,
			Name: b.Name}
	case *ast.TextBlock:
		res = &ast.TextBlock{
			Line: b.Line,
			Text: ctx.argsSubstText(m, args, opts, flags, b.Text)}
	}
	return res
}

func processUserMacro(exp Exporter, m *uMacroDefInfo) {
	ctx := exp.Context()
	// Do not allow too much depth
	if ctx.uMacroCall.depth > 42 {
		if ctx.Process {
			ctx.Error("recursive macro: too much depth (infinite recursive calls?)")
		}
		return
	}

	// curBlock: user defined macro
	if !ctx.Process {
		ctx.quiet = true
	}
	opts, flags, args := ctx.ParseOptions(m.opts, ctx.Args)
	if !ctx.Process {
		ctx.quiet = false
	}

	if !m.list && len(args) > m.argsc && ctx.Process {
		ctx.Error("too many arguments")
	}
	var blocks []ast.Block
	if m.argsc > 0 || m.list || len(m.opts) > 0 {
		// substitute $N arguments
		blocks = []ast.Block{}
		for _, b := range m.blocks {
			blocks = append(blocks, ctx.argsSubstBlock(m, args, opts, flags, b))
		}
	} else {
		blocks = m.blocks
	}

	// save user macro call location
	if ctx.uMacroCall.depth == 0 {
		ctx.uMacroCall.loc = ctx.loc
	}
	ctx.uMacroCall.depth++

	// process user macro blocks
	ctx.loc = &location{
		curBlock:  0,
		curBlocks: blocks,
		curFile:   m.file}
	processBlocks(exp)

	// recover location
	ctx.uMacroCall.depth--
	if ctx.uMacroCall.depth == 0 {
		ctx.loc = ctx.uMacroCall.loc
		ctx.uMacroCall.loc = nil
	}
}
