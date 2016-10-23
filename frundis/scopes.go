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
	inUserMacro bool
}

// pushScope adds a new scope
func (bctx *BaseContext) pushScope(s *scope) {
	st, ok := bctx.scopes[s.name]
	if !ok {
		st = []*scope{}
	}
	if bctx.callInfo.loc != nil {
		s.file = bctx.callInfo.loc.curFile
		b := bctx.callInfo.loc.curBlocks[bctx.callInfo.loc.curBlock]
		s.lnum = b.GetLine()
		s.inUserMacro = true
	} else {
		s.file = bctx.loc.curFile
		b := bctx.block()
		s.lnum = b.GetLine()
	}

	st = append(st, s)
	bctx.scopes[s.name] = st
}

// popScope pops a scope from specific tag
func (bctx *BaseContext) popScope(name string) *scope {
	st, ok := bctx.scopes[name]
	if !ok || len(st) == 0 {
		return nil
	}
	s := st[len(st)-1]
	bctx.scopes[name] = st[:len(st)-1]
	return s
}

func (bctx *BaseContext) scopeLocation(s *scope) string {
	var bfFile string
	if bctx.loc.curFile != "" {
		bfFile = " of file " + s.file
	}
	var inUserMacroMsg string
	if s.inUserMacro {
		inUserMacroMsg = "in user macro "
	}
	return fmt.Sprintf("%sat line %d%s", inUserMacroMsg, s.lnum, bfFile)
}
