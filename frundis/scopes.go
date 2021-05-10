// Scopes

package frundis

import "fmt"

type scope struct {
	tag         string
	tagRequired bool
	id          string
	lnum        int
	file        string
	name        string
	macro       string
	inUserMacro bool
}

// pushScope adds a new scope
func (ctx *Context) pushScope(s *scope) {
	st, ok := ctx.scopes[s.name]
	if !ok {
		st = []*scope{}
	}
	if ctx.uMacroCall.loc != nil {
		s.file = ctx.uMacroCall.loc.curFile
		b := ctx.uMacroCall.loc.curBlocks[ctx.uMacroCall.loc.curBlock]
		s.lnum = b.GetLine()
		s.inUserMacro = true
	} else {
		s.file = ctx.loc.curFile
		b := ctx.block()
		s.lnum = b.GetLine()
	}

	st = append(st, s)
	ctx.scopes[s.name] = st
}

// popScope pops a scope from specific tag
func (ctx *Context) popScope(name string) *scope {
	st, ok := ctx.scopes[name]
	if !ok || len(st) == 0 {
		return nil
	}
	s := st[len(st)-1]
	ctx.scopes[name] = st[:len(st)-1]
	return s
}

func (ctx *Context) scopeLocation(s *scope) string {
	var bfFile string
	if ctx.loc.curFile != "" {
		bfFile = " of file " + s.file
	}
	var inUserMacroMsg string
	if s.inUserMacro {
		inUserMacroMsg = "in user macro "
	}
	return fmt.Sprintf("%sat line %d%s", inUserMacroMsg, s.lnum, bfFile)
}
