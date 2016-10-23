// User macro processing stuff

package frundis

import (
	"fmt"

	"github.com/anaseto/gofrundis/ast"
)

// argsSubst returns inline text where numbered arguments are substituted with
// values from args
func (bctx *BaseContext) argsSubstText(
	args [][]ast.Inline, opts map[string][]ast.Inline,
	flags map[string]bool, text []ast.Inline) []ast.Inline {

	res := []ast.Inline{}
scanText:
	for _, elt := range text {
		switch elt := elt.(type) {
		case ast.ArgEscape:
			if int(elt) > len(args) || int(elt) <= 0 {
				bctx.Error("missing argument:$", elt)
				res = append(res, ast.Text(fmt.Sprint("\\$", int(elt))))
				continue scanText
			}
			res = append(res, args[elt-1]...)
		case ast.NamedArgEscape:
			arg, ok := opts[string(elt)]
			if !ok {
				bctx.Error("missing named argument:$[", elt, "]")
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
				for i, arg := range args {
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

func (bctx *BaseContext) argsSubstBlock(
	args [][]ast.Inline, opts map[string][]ast.Inline,
	flags map[string]bool, b ast.Block) ast.Block {

	var res ast.Block
	switch b := b.(type) {
	case *ast.Macro:
		nargs := [][]ast.Inline{}
		for _, arg := range b.Args {
			if len(arg) == 1 && arg[0] == ast.Escape("$@") {
				nargs = append(nargs, args...)
				continue
			}
			narg := bctx.argsSubstText(args, opts, flags, arg)
			nargs = append(nargs, narg)
		}
		res = &ast.Macro{
			Args: nargs,
			Line: b.Line,
			Name: b.Name}
	case *ast.TextBlock:
		res = &ast.TextBlock{
			Line: b.Line,
			Text: bctx.argsSubstText(args, opts, flags, b.Text)}
	}
	return res
}

func processUserMacro(exp BaseExporter) {
	bctx := exp.BaseContext()
	// Do not allow too much depth
	if bctx.callInfo.depth > 42 {
		bctx.Error("user macro invocation:too much depth (infinite recursive calls?)")
		return
	}

	// curBlock: user defined macro
	mb := bctx.block().(*ast.Macro)
	m, ok := bctx.macros[mb.Name]
	if !ok {
		bctx.Error("undefined macro:", mb.Name) // XXX useless (should not happen)
		return
	}
	opts, flags, args := bctx.parseOptions(m.opts, mb.Args)

	if len(args) > m.argsc {
		bctx.Error("too many arguments")
	}
	if len(m.opts) == 0 && (len(opts) > 0 || len(flags) > 0) {
		bctx.Error("unrecognized options")
	}
	var blocks []ast.Block
	if m.argsc > 0 || len(m.opts) > 0 {
		// substitute $N arguments
		blocks = []ast.Block{}
		for _, b := range m.blocks {
			blocks = append(blocks, bctx.argsSubstBlock(args, opts, flags, b))
		}
	} else {
		blocks = m.blocks
	}

	// save user macro call location
	if bctx.callInfo.depth == 0 {
		bctx.callInfo.loc = bctx.loc
	}
	bctx.callInfo.depth++

	// process user macro blocks
	bctx.loc = &location{
		curBlock:  0,
		curBlocks: blocks,
		curFile:   m.file}
	processBlocks(exp)

	// recover location
	bctx.callInfo.depth--
	if bctx.callInfo.depth == 0 {
		bctx.loc = bctx.callInfo.loc
		bctx.callInfo.loc = nil
	}
}
