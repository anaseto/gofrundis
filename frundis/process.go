// Block processing functions

package frundis

import (
	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/parser"
)

// ProcessFrundisSource processes a frundis file with a given exporter.
func ProcessFrundisSource(exp Exporter, filename string) error {
	exp.Init()
	err := processFile(exp, filename)
	if err != nil {
		return err
	}
	err = exp.Reset()
	if err != nil {
		return err
	}
	err = processFile(exp, filename)
	if err != nil {
		return err
	}
	bctx := exp.BaseContext()
	if bctx.loc == nil {
		bctx.loc = &location{curBlock: -1, curFile: filename}
	}
	bctx.Macro = "End Of File"
	closeUnclosedBlocks(exp, "Bm")
	closeUnclosedBlocks(exp, "Bl")
	closeUnclosedBlocks(exp, "Bd")
	testForUnclosedBlock(exp, "#if")
	_ = testForUnclosedFormatBlock(exp)
	testForUnclosedDe(exp)
	endParagraph(exp, false)
	exp.PostProcessing()
	return nil
}

// processFile does one pass through a file with a given exporter.
func processFile(exp BaseExporter, filename string) error {
	bctx := exp.BaseContext()
	blocks, ok := bctx.files[filename]
	if !ok {
		var err error
		p := parser.Parser{}
		blocks, err = p.ParseFile(filename)
		if err != nil {
			return err
		}
		bctx.files[filename] = blocks
	}
	loc := bctx.loc
	defer func() { bctx.loc = loc }()
	bctx.loc = &location{curBlocks: blocks, curFile: filename}
	processBlocks(exp)
	return nil
}

func processBlocks(exp BaseExporter) {
	bctx := exp.BaseContext()
	for i, b := range bctx.loc.curBlocks {
		bctx.loc.curBlock = i
		switch b := b.(type) {
		case *ast.Macro:
			bctx.Args = b.Args
			bctx.Macro = b.Name
			bctx.line = b.Line
		case *ast.TextBlock:
			bctx.text = b.Text
			bctx.line = b.Line
		}
		processBlock(exp)
	}
}

func processBlock(exp BaseExporter) {
	bctx := exp.BaseContext()
	b := bctx.block()
	if bctx.ifIgnore > 0 {
		b, ok := b.(*ast.Macro)
		if ok {
			switch b.Name {
			case "#;":
				macroIfEnd(exp)
			case "#if":
				macroIfStart(exp)
			default:
			}
		}
		return
	}
	if bctx.defInfo != nil {
		switch b := b.(type) {
		case *ast.Macro:
			switch b.Name {
			case "#.":
				macroDefEnd(exp)
			default:
				if !bctx.defInfo.ignore {
					bctx.defInfo.blocks = append(bctx.defInfo.blocks, b)
				}
			}
		case *ast.TextBlock:
			if !bctx.defInfo.ignore {
				bctx.defInfo.blocks = append(bctx.defInfo.blocks, b)
			}
		}
		return
	}

	if b, ok := b.(*ast.Macro); ok {
		_, ok = bctx.macros[b.Name]
		if ok {
			processUserMacro(exp)
			return
		}
		builtinHandler, ok := bctx.builtins[b.Name]
		if ok {
			builtinHandler(exp)
			bctx.PrevMacro = b.Name
			return
		}
	}
	exp.BlockHandler()
}

// DefaultExporterMacros returns a mapping from macros to handling functions,
// with the standard set of frundis macros.
func DefaultExporterMacros() map[string]func(Exporter) {
	return map[string]func(Exporter){
		"Bd": macroBd,
		"Bf": macroBf,
		"Bl": macroBl,
		"Bm": macroBm,
		"Ch": macroHeader,
		"D":  macroD,
		"Ed": macroEd,
		"Ef": macroEf,
		"El": macroEl,
		"Em": macroEm,
		"Ft": macroFt,
		"If": macroIncludeFile,
		"Im": macroIm,
		"It": macroIt,
		"Lk": macroLk,
		"P":  macroP,
		"Pt": macroHeader,
		"Sh": macroHeader,
		"Sm": macroSm,
		"Ss": macroHeader,
		"Sx": macroSx,
		"Ta": macroTa,
		"Tc": macroTc,
		"X":  macroX}
}

// MinimalExporterMacros returns a mapping from macros to handling functions,
// with only the following macros: Bd, Bf, Bm, Ed, Ef, Em, Ft, If, Sm, X.
func MinimalExporterMacros() map[string]func(Exporter) {
	return map[string]func(Exporter){
		"Bd": macroBd,
		"Bf": macroBf,
		"Bm": macroBm,
		"Ed": macroEd,
		"Ef": macroEf,
		"Em": macroEm,
		"Ft": macroFt,
		"If": macroIncludeFile,
		"Sm": macroSm,
		"X":  macroX}
}

// DefaultBlockHandler is as handler for Exporter method BlockHandler.
func DefaultBlockHandler(exp Exporter) {
	bctx := exp.BaseContext()
	b := bctx.block()
	switch b := b.(type) {
	case *ast.Macro:
		macros := exp.Context().Macros
		handler, ok := macros[b.Name]
		if ok {
			handler(exp)
		} else if b.Name != "" {
			bctx.Error("unknown macro:", b.Name)
		}
		bctx.PrevMacro = b.Name
	case *ast.TextBlock:
		doText(exp)
		bctx.PrevMacro = ""
	}
}
