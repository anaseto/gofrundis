// Block processing functions

package frundis

import (
	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/parser"
)

// ProcessFrundisSource processes a frundis file with a given exporter. In
// restricted mode, no #run nor shell filter are allowed.
func ProcessFrundisSource(exp Exporter, filename string, unrestricted bool) error {
	exp.Init()
	ctx := exp.Context()
	ctx.Unrestricted = unrestricted
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
	if ctx.loc == nil {
		ctx.loc = &location{curBlock: -1, curFile: filename}
	}
	ctx.Macro = "End Of File"
	closeUnclosedBlocks(exp, "Bm")
	closeUnclosedBlocks(exp, "Bl")
	closeUnclosedBlocks(exp, "Bd")
	checkForUnclosedBlock(exp, "#if")
	checkForUnclosedFormatBlock(exp)
	checkForUnclosedDe(exp)
	endParagraph(exp, ParBreakNormal)
	exp.PostProcessing()
	return nil
}

// processFile does one pass through a file with a given exporter.
func processFile(exp Exporter, filename string) error {
	ctx := exp.Context()
	blocks, ok := ctx.files[filename]
	if !ok {
		var err error
		p := parser.Parser{}
		blocks, err = p.ParseFile(filename)
		if err != nil {
			return err
		}
		ctx.files[filename] = blocks
	}
	loc := ctx.loc
	defer func() { ctx.loc = loc }()
	ctx.loc = &location{curBlocks: blocks, curFile: filename}
	processBlocks(exp)
	return nil
}

func processBlocks(exp Exporter) {
	ctx := exp.Context()
	for i, b := range ctx.loc.curBlocks {
		ctx.loc.curBlock = i
		switch b := b.(type) {
		case *ast.Macro:
			ctx.Args = b.Args
			ctx.Macro = b.Name
			ctx.line = b.Line
		case *ast.TextBlock:
			ctx.text = b.Text
			ctx.line = b.Line
		}
		processBlock(exp)
	}
}

func processBlock(exp Exporter) {
	ctx := exp.Context()
	b := ctx.block()
	if ctx.ifIgnoreDepth > 0 {
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
	if ctx.uMacroDef != nil {
		switch b := b.(type) {
		case *ast.Macro:
			switch b.Name {
			case "#.":
				macroDefEnd(exp)
			case "#de":
				macroDefStart(exp)
			default:
				if !ctx.uMacroDef.ignore {
					ctx.uMacroDef.blocks = append(ctx.uMacroDef.blocks, b)
				}
			}
		case *ast.TextBlock:
			if !ctx.uMacroDef.ignore {
				ctx.uMacroDef.blocks = append(ctx.uMacroDef.blocks, b)
			}
		}
		return
	}

	switch b := b.(type) {
	case *ast.Macro:
		m, ok := ctx.uMacros[b.Name]
		if ok {
			processUserMacro(exp, m)
			return
		}
		handler, ok := ctx.Macros[b.Name]
		if ok {
			if ctx.bfInfo != nil {
				switch b.Name {
				case "Ef", "#if", "#;":
				default:
					checkForUnclosedFormatBlock(exp)
				}
			}
			handler(exp)
			ctx.PrevMacro = b.Name
		} else if b.Name != "" && ctx.Process {
			ctx.Error("unknown macro:", b.Name)
		}
	case *ast.TextBlock:
		processText(exp)
		ctx.PrevMacro = ""
	}
}

// DefaultExporterMacros returns a mapping from macros to handling functions,
// with the standard set of frundis macros.
func DefaultExporterMacros() map[string]func(Exporter) {
	return map[string]func(Exporter){
		"Bd":   macroBd,
		"Bf":   macroBf,
		"Bl":   macroBl,
		"Bm":   macroBm,
		"Ch":   macroHeader,
		"D":    macroD,
		"Ed":   macroEd,
		"Ef":   macroEf,
		"El":   macroEl,
		"Em":   macroEm,
		"Ft":   macroFt,
		"If":   macroIncludeFile,
		"Im":   macroIm,
		"It":   macroIt,
		"Lk":   macroLk,
		"P":    macroP,
		"Pt":   macroHeader,
		"Sh":   macroHeader,
		"Sm":   macroSm,
		"Ss":   macroHeader,
		"Sx":   macroSx,
		"Ta":   macroTa,
		"Tc":   macroTc,
		"X":    macroX,
		"#de":  macroDefStart,
		"#.":   macroDefEnd,
		"#if":  macroIfStart,
		"#;":   macroIfEnd,
		"#dv":  macroDefVar,
		"#run": macroRun}
}

// MinimalExporterMacros returns a mapping from macros to handling functions,
// with only the following macros: Bd, Bf, Bm, Ed, Ef, Em, Ft, If, Sm, X.
func MinimalExporterMacros() map[string]func(Exporter) {
	return map[string]func(Exporter){
		"Bd":   macroBd,
		"Bf":   macroBf,
		"Bm":   macroBm,
		"Ed":   macroEd,
		"Ef":   macroEf,
		"Em":   macroEm,
		"Ft":   macroFt,
		"If":   macroIncludeFile,
		"Sm":   macroSm,
		"X":    macroX,
		"#de":  macroDefStart,
		"#.":   macroDefEnd,
		"#if":  macroIfStart,
		"#;":   macroIfEnd,
		"#dv":  macroDefVar,
		"#run": macroRun}
}
