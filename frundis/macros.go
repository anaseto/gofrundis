package frundis

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/anaseto/gofrundis/ast"
)

// text: as-is (not escaped), or regular (escaped + additional processing)

// -ns => no space even if wantspace

func doText(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	switch {
	case ctx.asIs:
		ctx.rawText.WriteString(bctx.InlinesToText(bctx.text))
	default:
		if !ctx.inpar {
			exp.BeginParagraph()
			ctx.inpar = true
			reopenSpanningBlocks(exp)
		} else if ctx.WantsSpace {
			// XXX: this can break tables for mom (and for markdown
			// things are not perfect either)
			fmt.Fprint(&ctx.buf, "\n")
		}
		if !ctx.inpar {
			ctx.inpar = true
		}
		text := exp.RenderText(bctx.text)
		if len(text) > 0 && hasBlankLine(text) {
			bctx.Error("empty line")
		}
		fmt.Fprint(&ctx.buf, text)
		ctx.WantsSpace = true
	}
}

// hasBlankLine returns true if string s has a line with only whitespace.
func hasBlankLine(s string) bool {
	blankline := true
	for _, c := range s {
		if c == '\n' {
			if blankline {
				return true
			}
			blankline = true
		} else if blankline && !unicode.IsSpace(c) {
			blankline = false
		}
	}
	return blankline
}

func macroBd(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	opts, flags, args := bctx.parseOptions(specOptBd, bctx.args)
	var id string
	if t, ok := opts["id"]; ok {
		id = exp.RenderText(t)
	}
	if !ctx.Process {
		if id != "" {
			ref := exp.GenRef("", id, false)
			if _, ok := ctx.IDs[id]; ok {
				bctx.Error("already used id")
			}
			ctx.IDs[id] = ref
		}
		return
	}
	if containsSpace(id) {
		bctx.Error("id identifier should not contain spaces")
	}
	if len(args) > 0 {
		bctx.Error("useless arguments")
	}
	closeUnclosedBlocks(exp, "Bm")
	closeUnclosedBlocks(exp, "Bl")

	var tag string
	if t, ok := opts["t"]; ok {
		tag = exp.RenderText(t)
	}

	softbreak := false
	if ctx.Dtags[tag].Cmd != "" {
		softbreak = true
	}
	endEventualParagraph(exp, softbreak)

	bctx.pushScope(&scope{name: "Bd", tag: tag, id: id, tagRequired: flags["r"]})

	if tag != "" {
		_, ok := ctx.Dtags[tag]
		if !ok {
			bctx.Error("invalid tag:", tag)
		}
	}
	exp.BeginDisplayBlock(tag, id)
}

func macroBf(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	opts, flags, args := bctx.parseOptions(specOptBf, bctx.args)
	if len(args) > 0 {
		bctx.Error("useless arguments")
	}
	fmt, okFmt := opts["f"]
	bfinf := bfInfo{line: bctx.line}
	ctx.bfInfo = &bfinf
	if bctx.callInfo.loc != nil {
		bfinf.file = bctx.callInfo.loc.curFile
		bfinf.inUserMacro = true
	} else {
		bfinf.file = bctx.loc.curFile
	}
	tag, okTag := opts["t"]
	ctx.asIs = true
	if !okFmt && !okTag {
		bctx.Error("you should specify a -f option or -t option at least")
		bfinf.ignore = true
		return
	}
	if okTag {
		tag := bctx.InlinesToText(tag)
		bfinf.filterTag = tag
		_, okGoFilter := ctx.Filters[tag]
		if !okGoFilter {
			bctx.Error("undefined filter tag '", tag)
			bfinf.ignore = true
			return
		}
	}
	if okFmt {
		formats := strings.Split(bctx.InlinesToText(fmt), ",")
		bctx.checkFormats(formats)
		if bctx.notExportFormat(formats) {
			bfinf.ignore = true
			return
		}
	}
	if ctx.inpar {
		beginPhrasingMacro(exp, flags["ns"])
		ctx.WantsSpace = false
	}
}

func macroBl(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		macroBlProcess(exp)
	} else {
		macroBlInfos(exp)
	}
}

func macroBlInfos(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	opts, _, args := bctx.parseOptions(specOptBl, bctx.args)
	tag, ok := opts["t"]
	if !ok {
		return
	}
	switch bctx.InlinesToText(tag) {
	case "verse":
		ctx.HasVerse = true
		title := renderArgs(exp, args)
		if title == "" {
			return
		}
		titleText := processInlineMacros(exp, args)
		ctx.verseCount++
		loXEntryInfos(exp, "lop",
			&LoXinfo{
				Title:     title,
				TitleText: titleText,
				Count:     ctx.verseCount,
				RefPrefix: "poem"},
			strconv.Itoa(ctx.verseCount))
	case "table":
		ctx.tableIn = true
		title := renderArgs(exp, args)
		if title == "" {
			return
		}
		titleText := processInlineMacros(exp, args)
		ctx.TableCount++
		ctx.tableScope = true
		loXEntryInfos(exp, "lot",
			&LoXinfo{
				Title:     title,
				TitleText: titleText,
				Count:     ctx.TableCount,
				RefPrefix: "tbl"},
			strconv.Itoa(ctx.TableCount))
	}
}

func macroBlProcess(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	opts, _, args := bctx.parseOptions(specOptBl, bctx.args)
	var tag string
	if t, ok := opts["t"]; ok {
		tag = bctx.InlinesToText(t)
	} else {
		tag = "item"
	}
	switch tag {
	case "item", "enum", "desc", "verse", "table":
		// Ok, do nothing
	default:
		bctx.Error("invalid `-t' option argument:", tag)
		tag = "item" // fallback to basic "item" list
	}
	scopes, ok := bctx.scopes["Bl"]
	if ok && len(scopes) > 0 {
		last := scopes[len(scopes)-1]
		if last == nil || last.tag != "item" && last.tag != "enum" {
			bctx.Error("nested list of invalid type")
			return
		}
		if ctx.inpar {
			parEnd(exp)
		}
	} else {
		endEventualParagraph(exp, true)
	}

	bctx.pushScope(&scope{name: "Bl", tag: tag})

	switch tag {
	case "verse":
		title := processInlineMacros(exp, args)
		if title != "" {
			ctx.verseCount++
		}
		exp.BeginVerse(title, ctx.verseCount)
	case "desc":
		exp.BeginDescList()
	case "item":
		exp.BeginItemList()
	case "enum":
		exp.BeginEnumList()
	case "table":
		title := processInlineMacros(exp, args)
		ctx.tableIn = true
		if title != "" {
			ctx.TableCount++
			ctx.tableScope = true
		}
		exp.BeginTable(title, ctx.TableCount, ctx.tableInfo[ctx.TableNum].Cols)
	}
	ctx.itemScope = false
}

func macroBm(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	opts, flags, args := bctx.parseOptions(specOptBm, bctx.args)
	var id string
	if t, ok := opts["id"]; ok {
		id = exp.RenderText(t)
	}
	if !ctx.Process {
		if id != "" {
			ref := exp.GenRef("", id, false)
			if _, ok := ctx.IDs[id]; ok {
				bctx.Error("already used id")
			}
			ctx.IDs[id] = ref
		}
		return
	}

	beginPhrasingMacro(exp, flags["ns"])
	ctx.WantsSpace = false
	var tag string
	if t, ok := opts["t"]; ok {
		tag = bctx.InlinesToText(t)
		_, ok := ctx.Mtags[tag]
		if !ok {
			bctx.Error("invalid tag argument to `-t' option")
		}
	}
	bctx.pushScope(&scope{name: "Bm", tag: tag, id: id, tagRequired: flags["r"]})
	exp.BeginMarkupBlock(tag, id)
	if len(args) > 0 {
		if !ctx.Inline {
			bctx.Error("useless arguments")
		} else {
			w := ctx.GetW()
			fmt.Fprint(w, renderArgs(exp, args))
		}
	}
}

func macroD(exp Exporter) {
	bctx := exp.BaseContext()
	_, _, args := bctx.parseOptions(specOptD, bctx.args)
	if len(args) > 0 {
		bctx.Error("useless arguments")
	}
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	if ctx.inpar {
		closeSpanningBlocks(exp)
		parEnd(exp)
		exp.EndParagraph()
	}
	exp.BeginParagraph()
	ctx.inpar = true
	reopenSpanningBlocks(exp)
	exp.BeginDialogue()
	ctx.WantsSpace = false
}

func macroEd(exp Exporter) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	if !ctx.Process {
		return
	}
	opts, _, args := bctx.parseOptions(specOptEd, bctx.args)
	if len(args) > 0 {
		bctx.Error("useless arguments")
	}
	scope := bctx.popScope("Bd")
	if scope == nil {
		bctx.Error("no corresponding `.Bd'")
		return
	}
	if tag, ok := opts["t"]; ok {
		if bctx.InlinesToText(tag) != scope.tag {
			location := bctx.scopeLocation(scope)
			bctx.Error("tag doesn't match tag '", scope.tag, "' of current block opened ", location)
		}
	} else if scope.tagRequired {
		location := bctx.scopeLocation(scope)
		bctx.Error("missing required tag matching tag '", scope.tag, "' of current block opened ", location)
	}
	softbreak := false
	if ctx.Dtags[scope.tag].Cmd != "" {
		softbreak = true
	}
	endEventualParagraph(exp, softbreak)
	exp.EndDisplayBlock(scope.tag)

	ctx.WantsSpace = false
}

func macroEf(exp Exporter) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	if !ctx.Process {
		return
	}
	_, flags, args := bctx.parseOptions(specOptEf, bctx.args)
	if len(args) > 0 {
		bctx.Error("useless arguments")
	}
	if ctx.bfInfo == nil {
		bctx.Error("no corresponding `.Bf'")
		return
	}
	if !ctx.bfInfo.ignore {
		var text string
		if tag := ctx.bfInfo.filterTag; tag != "" {
			filter, ok := ctx.Filters[tag]
			if ok {
				text = filter(ctx.rawText.String())
			} else {
				bctx.Error("invalid filter tag:", tag) // XXX improve message
				text = ctx.rawText.String()
			}
		} else {
			text = ctx.rawText.String()
		}
		w := ctx.GetW()
		fmt.Fprint(w, text)
		if ctx.inpar && !flags["ns"] {
			ctx.WantsSpace = true
		} else {
			fmt.Fprint(w, "\n")
		}
	}
	ctx.rawText.Reset()
	ctx.asIs = false
	ctx.bfInfo = nil
}

func macroEl(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		macroElInfos(exp)
	} else {
		macroElProcess(exp)
	}
}

func macroElInfos(exp Exporter) {
	ctx := exp.Context()
	if ctx.tableIn {
		if ctx.TableCols == 0 {
			ctx.TableCols = ctx.TableCell
		}
		ctx.tableInfo = append(ctx.tableInfo, &TableInfo{Cols: ctx.TableCols})
		ctx.tableIn = false
		ctx.tableScope = false
		ctx.TableCell = 0
		ctx.TableCols = 0
		ctx.TableNum++
	}
}

func macroElProcess(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	scope := bctx.popScope("Bl")
	if scope == nil {
		bctx.Error("no corresponding `.Bl'")
		return
	}
	_, _, args := bctx.parseOptions(specOptEl, bctx.args)
	if len(args) > 0 {
		bctx.Error("useless arguments")
	}
	if !ctx.itemScope {
		switch scope.tag {
		case "desc":
			bctx.Error("no previous `.It' in 'desc' list. Empty list?")
			exp.BeginDescValue()
		case "item":
			bctx.Error("no previous `.It'. Empty list?")
			exp.BeginItem()
		case "enum":
			bctx.Error("no previous `.It'. Empty list?")
			exp.BeginEnumItem()
		default:
			if ctx.inpar {
				bctx.Error("unexpected accumulated text outside item scope")
			}
		}
	}

	switch scope.tag {
	case "verse":
		parEnd(exp)
		exp.EndParagraph()
		exp.EndVerse()
	case "desc":
		parEnd(exp)
		exp.EndDescValue()
		exp.EndDescList()
	case "enum":
		parEnd(exp)
		exp.EndEnumItem()
		exp.EndEnumList()
	case "item":
		parEnd(exp)
		exp.EndItem()
		exp.EndItemList()
	case "table":
		// allow empty table
		if ctx.itemScope {
			parEnd(exp)
			exp.EndTableCell()
			exp.EndTableRow()
		}
		var tableinfo *TableInfo
		if ctx.tableScope {
			info, ok := ctx.LoXstack["lot"]
			if ok {
				tableinfo = &TableInfo{Title: info[ctx.TableCount-1].TitleText}
			} else {
				tableinfo = &TableInfo{}
				bctx.Error("internal error about table info")
			}
		}
		exp.EndTable(tableinfo)
		ctx.tableScope = false
		ctx.tableIn = false
		ctx.TableCell = 0
		ctx.TableCols = 0
		ctx.TableNum++
	}
	scopes, ok := bctx.scopes["Bl"]
	if ok && len(scopes) > 0 {
		ctx.itemScope = true
		ctx.inpar = true
	} else {
		ctx.itemScope = false
	}
	ctx.WantsSpace = false
}

func macroEm(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	opts, _, args := bctx.parseOptions(specOptEm, bctx.args)
	scope := bctx.popScope("Bm")
	if scope == nil {
		bctx.Error("no corresponding `.Bm'")
		return
	}
	if tag, ok := opts["t"]; ok {
		if bctx.InlinesToText(tag) != scope.tag {
			location := bctx.scopeLocation(scope)
			bctx.Error("tag doesn't match tag '", scope.tag, "' of current block opened ", location)
		}
	} else if scope.tagRequired {
		location := bctx.scopeLocation(scope)
		bctx.Error("missing required tag matching tag '", scope.tag, "' of current block opened ", location)
	}
	tag := scope.tag
	id := scope.id
	var punct string
	if len(args) > 0 {
		if !ctx.Inline || bctx.isPunctArg(args[0]) {
			punct = exp.RenderText(args[0])
			args = args[1:]
		}
	}
	exp.EndMarkupBlock(tag, id, punct)
	if len(args) > 0 {
		if !ctx.Inline {
			bctx.Error("useless args in macro `.Em'")
		} else {
			w := ctx.GetW()
			fmt.Fprint(w, renderArgs(exp, args))
		}
	}
	ctx.WantsSpace = true
}

func macroFt(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	opts, flags, args := bctx.parseOptions(specOptFt, bctx.args)
	format, okFmt := opts["f"]
	if okFmt {
		formats := strings.Split(bctx.InlinesToText(format), ",")
		bctx.checkFormats(formats)
		if bctx.notExportFormat(formats) {
			return
		}
	}
	tag, okTag := opts["t"]
	if !okFmt && !okTag {
		bctx.Error("you should specify a -f option or -t option at least")
		return
	}
	scopes, okScope := bctx.scopes["Bl"]
	if okScope && len(scopes) > 0 && !ctx.itemScope {
		bctx.Error("invocation in `.Bl' list outside `.It' scope")
		return
	}
	if ctx.inpar {
		beginPhrasingMacro(exp, flags["ns"])
		ctx.WantsSpace = false
	}
	// If ctx.Buf is empty, we write directly to ctx.W
	var text string
	if okTag {
		tag := bctx.InlinesToText(tag)
		goFilter, okGoFilter := ctx.Filters[tag]
		if okGoFilter {
			text = goFilter(argsToText(exp, args, " "))
		} else {
			bctx.Error("undefined filter tag '", tag)
			text = renderArgs(exp, args)
		}
	} else {
		text = argsToText(exp, args, " ")
	}
	w := ctx.GetW()
	fmt.Fprint(w, text)
}

func macroIncludeFile(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	opts, flags, args := bctx.parseOptions(specOptIncludeFile, bctx.args)
	if format, ok := opts["f"]; ok {
		formats := strings.Split(bctx.InlinesToText(format), ",")
		bctx.checkFormats(formats)
		if bctx.notExportFormat(formats) {
			return
		}
	}
	if len(args) == 0 {
		if ctx.Process {
			bctx.Error("filename argument required")
		}
		return
	}
	filename := bctx.InlinesToText(args[0])
	if flags["as-is"] {
		if !ctx.Process {
			return
		}
		if ctx.inpar {
			beginPhrasingMacro(exp, flags["ns"])
			ctx.WantsSpace = true
		}
		source, err := ioutil.ReadFile(filename)
		if err != nil {
			bctx.Error("as-is inclusion:", err)
			return
		}
		var text string
		if t, ok := opts["t"]; ok {
			tag := bctx.InlinesToText(t)
			if filter, ok := ctx.Filters[tag]; ok {
				text = filter(string(source))
			} else {
				text = string(source)
				bctx.Error("unknown tag:", tag)
			}
		} else {
			text = string(source)
		}
		w := ctx.GetW()
		fmt.Fprint(w, text)
	} else {
		// frundis source file
		filename, ok := SearchIncFile(exp, filename)
		if !ok {
			bctx.Error("no such frundis source file")
			return
		}
		err := processFile(exp, filename)
		if err != nil {
			bctx.Error(err)
		}
	}
}

func macroIm(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		macroImProcess(exp)
	} else {
		macroImInfos(exp)
	}
}

func macroImProcess(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	opts, flags, args := bctx.parseOptions(specOptIm, bctx.args)
	args, punct := getClosePunct(exp, args)
	var link string
	if t, ok := opts["link"]; ok {
		link = bctx.InlinesToText(t)
	}
	if len(args) > 2 {
		bctx.Error("too many arguments")
		args = args[:2]
	}
	switch len(args) {
	case 0:
		bctx.Error("requires at least one argument")
	case 1:
		beginPhrasingMacro(exp, flags["ns"])
		ctx.WantsSpace = true
		image := bctx.InlinesToText(args[0])
		exp.InlineImage(image, link, punct)
	case 2:
		closeUnclosedBlocks(exp, "Bm")
		closeUnclosedBlocks(exp, "Bl")
		endEventualParagraph(exp, false)
		image := bctx.InlinesToText(args[0])
		label := exp.RenderText(args[1])
		ctx.FigCount++
		exp.FigureImage(image, label, link)
	}
}

func macroImInfos(exp Exporter) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	_, _, args := bctx.parseOptions(specOptIm, bctx.args)
	args, _ = getClosePunct(exp, args)
	var image string
	if len(args) > 0 {
		ctx.HasImage = true
		image = bctx.InlinesToText(args[0])
		ctx.Images = append(ctx.Images, image)
	}
	if len(args) > 1 {
		ctx.HasImage = true
		label := exp.RenderText(args[1])
		ctx.FigCount++
		loXEntryInfos(exp, "lof",
			&LoXinfo{
				Title:     label,
				TitleText: label,
				Count:     ctx.FigCount,
				RefPrefix: "fig"},
			strconv.Itoa(ctx.FigCount))
	}
}

func macroIt(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		macroItProcess(exp)
	} else {
		macroItInfos(exp)
	}
}

func macroItInfos(exp Exporter) {
	ctx := exp.Context()
	if ctx.tableIn {
		if ctx.TableCols == 0 {
			ctx.TableCols = ctx.TableCell
		}
		ctx.TableCell = 1
	}
}

func macroItProcess(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	_, _, args := bctx.parseOptions(specOptIt, bctx.args)
	scopes, ok := bctx.scopes["Bl"]
	if !ok || len(scopes) == 0 {
		bctx.Error("outside `.Bl' macro scope")
		return
	}
	closeUnclosedBlocks(exp, "Bm")
	scope := scopes[len(scopes)-1]
	ctx.WantsSpace = false
	switch scope.tag {
	case "desc":
		macroItDesc(exp, args)
	case "item", "enum":
		macroItemenum(exp, args, scope.tag)
	case "table":
		macroItTable(exp, args)
	case "verse":
		macroItVerse(exp, args)
	}
	ctx.itemScope = true
}

func macroItDesc(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	if ctx.itemScope {
		parEnd(exp)
		exp.EndDescValue()
	}
	if len(args) == 0 {
		bctx.Error("description name required")
	}
	name := processInlineMacros(exp, args)
	ctx.WantsSpace = false
	exp.DescName(name)
	exp.BeginDescValue()
	ctx.inpar = true
}

func macroItemenum(exp Exporter, args [][]ast.Inline, tag string) {
	ctx := exp.Context()
	if ctx.itemScope {
		parEnd(exp)
		switch tag {
		case "item":
			exp.EndItem()
		case "enum":
			exp.EndEnumItem()
		}
	}
	switch tag {
	case "item":
		exp.BeginItem()
	case "enum":
		exp.BeginEnumItem()
	}
	ctx.inpar = true
	ctx.WantsSpace = false
	if len(args) > 0 {
		w := ctx.GetW()
		fmt.Fprint(w, processInlineMacros(exp, args))
		ctx.WantsSpace = true
	}
}

func macroItTable(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	if ctx.itemScope {
		parEnd(exp)
		exp.EndTableCell()
		exp.EndTableRow()
	}
	if ctx.TableCols == 0 {
		ctx.TableCols = ctx.TableCell
	}
	if ctx.TableCols > ctx.TableCell {
		bctx.Error("not enough cells in previous row")
	}
	ctx.TableCell = 1
	exp.BeginTableRow()
	exp.BeginTableCell()
	ctx.inpar = true
	if len(args) > 0 {
		w := ctx.GetW()
		fmt.Fprint(w, processInlineMacros(exp, args))
		ctx.WantsSpace = true
	}
}

func macroItVerse(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	if !ctx.inpar {
		exp.BeginParagraph()
		ctx.inpar = true
	} else if ctx.itemScope {
		exp.EndVerseLine()
	}
	if len(args) > 0 {
		w := ctx.GetW()
		fmt.Fprint(w, processInlineMacros(exp, args))
		ctx.WantsSpace = true
	}
}

func macroLk(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	_, flags, args := bctx.parseOptions(specOptLk, bctx.args)
	var punct string
	if len(args) > 1 {
		args, punct = getClosePunct(exp, args)
	}
	if len(args) == 0 {
		bctx.Error("argument required")
		return
	}
	beginPhrasingMacro(exp, flags["ns"])
	ctx.WantsSpace = true

	if len(args) >= 2 {
		if len(args) > 2 {
			bctx.Error("too many arguments")
		}
		url := bctx.InlinesToText(args[0])
		label := exp.RenderText(args[1])
		exp.LkWithLabel(url, label, punct)
	} else {
		url := bctx.InlinesToText(args[0])
		exp.LkWithoutLabel(url, punct)
	}
}

func macroP(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	bctx := exp.BaseContext()
	_, _, args := bctx.parseOptions(specOptP, bctx.args)
	if ctx.inpar {
		closeSpanningBlocks(exp)
		parEnd(exp)
		exp.EndParagraph()
	} else {
		exp.EndParagraphUnsoftly()
		ctx.inpar = false
	}
	if len(args) > 0 {
		ctx.inpar = true
		title := processInlineMacros(exp, args)
		exp.ParagraphTitle(title)
		reopenSpanningBlocks(exp)
	}
	ctx.WantsSpace = false
	ctx.itemScope = false // for verse
}

func macroSm(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	opts, flags, args := bctx.parseOptions(specOptSm, bctx.args)
	var id string
	if t, ok := opts["id"]; ok {
		id = exp.RenderText(t)
	}
	if !ctx.Process {
		if id != "" {
			ref := exp.GenRef("", id, false)
			if _, ok := ctx.IDs[id]; ok {
				bctx.Error("already used id")
			}
			ctx.IDs[id] = ref
		}
		return
	}
	if len(args) == 0 {
		bctx.Error("arguments required")
		return
	}
	var punct string
	if len(args) > 1 {
		args, punct = getClosePunct(exp, args)
	}

	beginPhrasingMacro(exp, flags["ns"])
	var tag string
	if t, ok := opts["t"]; ok {
		tag = bctx.InlinesToText(t)
		_, ok := ctx.Mtags[tag]
		if !ok {
			bctx.Error("invalid tag argument to `-t' option")
		}
	}
	exp.BeginMarkupBlock(tag, id)
	w := ctx.GetW()
	fmt.Fprint(w, renderArgs(exp, args))
	exp.EndMarkupBlock(tag, id, punct)
	ctx.WantsSpace = true
}

func macroSx(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	bctx := exp.BaseContext()
	opts, flags, args := bctx.parseOptions(specOptSx, bctx.args)
	tag := "toc" // default value
	if t, ok := opts["t"]; ok {
		tag = bctx.InlinesToText(t)
	}
	var punct string
	if len(args) > 1 {
		args, punct = getClosePunct(exp, args)
	}
	if len(args) == 0 {
		bctx.Error("arguments required")
		return
	}
	loX, okloX := ctx.LoXInfo[tag]
	if !okloX && !flags["id"] {
		bctx.Error("invalid argument to -type:", tag)
		return
	}
	id := renderArgs(exp, args)
	var loXentry *LoXinfo
	if !flags["id"] {
		entry, ok := loX[id]
		if ok {
			loXentry = entry
		} else {
			bctx.Error("unknown title for type '", tag, "':", id)
			id = ""
		}
	}
	beginPhrasingMacro(exp, flags["ns"])
	ctx.WantsSpace = true
	var name string
	if t, ok := opts["name"]; ok {
		name = exp.RenderText(t)
	} else {
		name = processInlineMacros(exp, args)
	}
	if flags["id"] {
		_, ok := ctx.IDs[id]
		if !ok {
			bctx.Error("reference to unknown id '", id, "'")
			id = ""
		}
	}
	exp.CrossReference(id, name, loXentry, punct)
}

func macroTa(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		macroTaProcess(exp)
	} else {
		macroTaInfos(exp)
	}
}

func macroTaInfos(exp Exporter) {
	ctx := exp.Context()
	ctx.TableCell++
}

func macroTaProcess(exp Exporter) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	_, _, args := bctx.parseOptions(specOptTa, bctx.args)
	scopes, hasBl := bctx.scopes["Bl"]
	if !hasBl || len(scopes) == 0 {
		bctx.Error("outside `.Bl -t table' scope")
		return
	}
	scope := scopes[len(scopes)-1]
	if scope.tag != "table" {
		bctx.Error("not a ``table'' list")
		return
	}
	if !ctx.itemScope {
		bctx.Error("outside an `.It' row scope")
		return
	}
	closeUnclosedBlocks(exp, "Bm")
	parEnd(exp)
	exp.EndTableCell()
	ctx.TableCell++
	exp.BeginTableCell()
	ctx.inpar = true
	if len(args) > 0 {
		w := ctx.GetW()
		fmt.Fprint(w, processInlineMacros(exp, args))
		ctx.WantsSpace = true
	} else {
		ctx.WantsSpace = false
	}
}

func macroTc(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		macroTcProcess(exp)
	} else {
		macroTcInfos(exp)
	}
}

func macroTcInfos(exp Exporter) {
	bctx := exp.BaseContext()
	_, flags, _ := bctx.parseOptions(specOptTc, bctx.args)
	exp.TableOfContentsInfos(flags)
}

func macroTcProcess(exp Exporter) {
	closeUnclosedBlocks(exp, "Bm")
	closeUnclosedBlocks(exp, "Bl")
	bctx := exp.BaseContext()
	opts, flags, args := bctx.parseOptions(specOptTc, bctx.args)
	if len(args) > 0 {
		bctx.Error("useless arguments")
	}
	endEventualParagraph(exp, flags["ns"])
	var toc, lof, lot, lop = flags["toc"], flags["lof"], flags["lot"], flags["lop"]
	if !(toc || lof || lot || lop) {
		toc = true
		if flags == nil {
			flags = make(map[string]bool)
		}
		flags["toc"] = true
	}
	loXtypes := []bool{toc, lof, lot, lop}
	count := 0
	for _, t := range loXtypes {
		if t {
			count++
		}
	}
	if count > 1 {
		bctx.Error("only one of the -toc, -lof and -lot options should bet set")
		return
	}
	exp.TableOfContents(opts, flags)
}

func macroX(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		return
	}
	bctx := exp.BaseContext()
	args := bctx.args
	if len(args) == 0 {
		bctx.Error("you should specify arguments")
		return
	}
	cmd := bctx.InlinesToText(args[0])
	args = args[1:]
	bctx.Macro += " " + cmd
	switch cmd {
	case "dtag":
		macroXdtag(exp, args)
	case "ftag":
		macroXftag(exp, args)
	case "mtag":
		macroXmtag(exp, args)
	case "set":
		macroXset(exp, args)
	}
}

func macroXdtag(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	var opts map[string][]ast.Inline
	opts, _, args = bctx.parseOptions(specOptXdtag, args)
	var formats []string
	if format, okFmt := opts["f"]; okFmt {
		formats = strings.Split(bctx.InlinesToText(format), ",")
	} else {
		bctx.Error("you should specify `-f' option")
		return
	}
	bctx.checkFormats(formats)
	if bctx.notExportFormat(formats) {
		return
	}
	var tag string
	if t, ok := opts["t"]; ok {
		tag = bctx.InlinesToText(t)
		if tag == "" {
			bctx.Error("tag option argument cannot be empty")
			return
		}
	} else {
		bctx.Error("-t option should be specified")
		return
	}
	cmd := bctx.InlinesToText(opts["c"])
	ctx.Dtags[tag] = exp.Xdtag(cmd)
}

func macroXftag(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	var opts map[string][]ast.Inline
	opts, _, args = bctx.parseOptions(specOptXftag, args)
	if format, ok := opts["f"]; ok {
		formats := strings.Split(bctx.InlinesToText(format), ",")
		bctx.checkFormats(formats)
		if len(formats) > 0 && bctx.notExportFormat(formats) {
			return
		}
	}
	var tag string
	if t, ok := opts["t"]; ok {
		tag = bctx.InlinesToText(t)
		if tag == "" {
			bctx.Error("tag option argument cannot be empty")
			return
		}
	} else {
		bctx.Error("-t option should be specified")
		return
	}
	if t, ok := opts["shell"]; ok {
		shell := bctx.InlinesToText(t)
		ctx.Filters[tag] = func(text string) string { return shellFilter(exp, shell, text) }
		return
	}
	if t, ok := opts["gsub"]; ok {
		s := bctx.InlinesToText(t)
		sr := strings.NewReader(s)
		r, size, err := sr.ReadRune()
		if err != nil {
			bctx.Error("invalid -gsub argument")
			return
		}
		s = s[size:]
		repls := strings.Split(s, fmt.Sprintf("%c", r))
		if len(repls)%2 != 0 {
			bctx.Error("invalid -gsub argument (non even number of strings)")
			return
		}
		escaper := strings.NewReplacer(repls...)
		ctx.Filters[tag] = func(text string) string { return escaper.Replace(text) }
		return
	}
	if t, ok := opts["regexp"]; ok {
		s := bctx.InlinesToText(t)
		sr := strings.NewReader(s)
		r, size, err := sr.ReadRune()
		if err != nil {
			bctx.Error("invalid -regexp argument")
			return
		}
		s = s[size:]
		repls := strings.Split(s, fmt.Sprintf("%c", r))
		if len(repls) != 2 {
			bctx.Error("invalid -regexp argument (missing separator?)")
			return
		}
		rx, err := regexp.Compile(repls[0])
		if err != nil {
			bctx.Error("invalid -regexp argument:", err)
			return
		}
		ctx.Filters[tag] = func(text string) string { return rx.ReplaceAllString(text, repls[1]) }
		return
	}

	bctx.Error("one of -shell/-gsub/-regexp option should be provided")
}

func macroXmtag(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	var opts map[string][]ast.Inline
	opts, _, args = bctx.parseOptions(specOptXmtag, args)
	var formats []string
	if format, okFmt := opts["f"]; okFmt {
		formats = strings.Split(bctx.InlinesToText(format), ",")
	} else {
		bctx.Error("you should specify `-f' option")
		return
	}
	bctx.checkFormats(formats)
	if bctx.notExportFormat(formats) {
		return
	}
	var tag string
	if t, ok := opts["t"]; ok {
		tag = bctx.InlinesToText(t)
		if tag == "" {
			bctx.Error("tag option argument cannot be empty")
			return
		}
	} else {
		bctx.Error("-t option should be specified")
		return
	}
	b, e := bctx.InlinesToText(opts["b"]), bctx.InlinesToText(opts["e"])
	var cmd *string
	if t, ok := opts["c"]; ok {
		s := bctx.InlinesToText(t)
		if s == "" {
			bctx.Error("empty string argument to -c option")
		}
		cmd = &s
	}
	var pairs []string
	if t, ok := opts["a"]; ok {
		s := bctx.InlinesToText(t)
		var err error
		pairs, err = readPairs(s)
		if err != nil {
			bctx.Error("invalid -a argument (missing separator?)")
		}
		for i := 0; i < len(pairs)-1; i += 2 {
			if pairs[i] == "" {
				bctx.Error(fmt.Sprintf("key %d is empty in -a option", (i/2)+1))
			}
		}
	}
	ctx.Mtags[tag] = exp.Xmtag(cmd, b, e, pairs)
}

func macroXset(exp Exporter, args [][]ast.Inline) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	var opts map[string][]ast.Inline
	opts, _, args = bctx.parseOptions(specOptXset, args)
	var formats []string
	if format, okFmt := opts["f"]; okFmt {
		formats = strings.Split(bctx.InlinesToText(format), ",")
	}
	bctx.checkFormats(formats)
	if len(formats) > 0 && bctx.notExportFormat(formats) {
		return
	}
	if len(args) < 2 {
		bctx.Error("two arguments expected")
		return
	}
	if len(args) > 2 {
		bctx.Error("too many arguments")
	}
	param := bctx.InlinesToText(args[0])
	value := bctx.InlinesToText(args[1])
	switch param {
	case "dmark", "document-author", "document-date", "document-title",
		"epub-cover", "epub-css", "epub-metadata", "epub-subject", "epub-uuid", "epub-version",
		"lang",
		"latex-preamble", "latex-xelatex",
		"mom-preamble",
		"nbsp", "title-page",
		"xhtml-bottom", "xhtml-css", "xhtml-index", "xhtml-go-up", "xhtml-top", "xhtml5":
		// XXX use map for this check ?
	default:
		bctx.Error("unknown parameter:", param)
	}
	if exp.CheckParamAssignement(param, value) {
		ctx.Params[param] = value
	}
}

func macroHeader(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		macroHeaderProcess(exp)
	} else {
		macroHeaderInfos(exp)
	}
}

func macroHeaderProcess(exp Exporter) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	_, flags, args := bctx.parseOptions(specOptHeader, bctx.args)
	if len(args) == 0 {
		bctx.Error("arguments required")
		return
	}
	numbered := !flags["nonum"]
	title := renderArgs(exp, args)
	closeUnclosedBlocks(exp, "Bm")
	closeUnclosedBlocks(exp, "Bl")
	endEventualParagraph(exp, false)
	ctx.TocInfo.updateHeadersCount(bctx.Macro, flags["nonum"])
	titleText := processInlineMacros(exp, args)
	exp.BeginHeader(bctx.Macro, title, numbered, titleText)
	fmt.Fprint(ctx.GetW(), titleText)
	closeUnclosedBlocks(exp, "Bm")
	exp.EndHeader(bctx.Macro, title, numbered, titleText)
}

func macroHeaderInfos(exp Exporter) {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	_, flags, args := bctx.parseOptions(specOptHeader, bctx.args)
	if len(args) == 0 {
		// Error message while processing
		return
	}
	ctx.TocInfo.updateHeadersCount(bctx.Macro, flags["nonum"])
	switch bctx.Macro {
	case "Pt":
		ctx.TocInfo.HasPart = true
	case "Ch":
		ctx.TocInfo.HasChapter = true
	}
	ref := exp.HeaderReference(bctx.Macro)
	title := renderArgs(exp, args)
	tocInfo, ok := ctx.LoXInfo["toc"]
	if !ok {
		ctx.LoXInfo["toc"] = make(map[string]*LoXinfo)
		tocInfo = ctx.LoXInfo["toc"]
	}
	num := ctx.TocInfo.HeaderNum(bctx.Macro, flags["nonum"])
	titleText := processInlineMacros(exp, args)
	tocInfo[title] = &LoXinfo{
		Count:     ctx.TocInfo.HeaderCount,
		Ref:       ref,
		RefPrefix: "s",
		Macro:     bctx.Macro,
		Nonum:     flags["nonum"],
		Title:     title,
		TitleText: titleText,
		Num:       num}
	switch bctx.Macro {
	case "Pt", "Ch":
		ctx.LoXstack["nav"] = append(ctx.LoXstack["nav"], tocInfo[title])
	}
	ctx.LoXstack["toc"] = append(ctx.LoXstack["toc"], tocInfo[title])
}

// processInlineMacros processes a list of arguments with Sm-like markup and
// returns the result.
func processInlineMacros(exp Exporter, args [][]ast.Inline) string {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	oldBuf := ctx.buf
	ctx.buf = bytes.Buffer{}
	defer func() {
		ctx.buf = oldBuf
	}()
	blocks := []ast.Block{}
	for _, arg := range args {
		if len(arg) == 0 {
			continue
		}
		switch arg[0] {
		case ast.Text("Bm"), ast.Text("Em"), ast.Text("Sm"):
			if len(arg) == 1 {
				blocks = append(blocks,
					&ast.Macro{
						Args: [][]ast.Inline{},
						Line: bctx.line,
						Name: bctx.inlineToText(arg[0])})
				break
			}
			fallthrough
		default:
			if len(blocks) == 0 {
				// No Sm or Bm as first argument: initialize text block
				blocks = append(blocks, &ast.TextBlock{Line: bctx.line})
			}
			b := blocks[len(blocks)-1]
			switch b := b.(type) {
			case *ast.Macro:
				b.Args = append(b.Args, arg)
			case *ast.TextBlock:
				if len(b.Text) > 0 {
					b.Text = append(b.Text, ast.Text(" "))
				}
				for _, elt := range arg {
					b.Text = append(b.Text, elt)
				}
			}
		}
	}
	loc := bctx.loc
	curMacro := bctx.Macro
	curArgs := bctx.args
	ws := ctx.WantsSpace
	oldpar := ctx.inpar
	proc := ctx.Process
	ctx.WantsSpace = false
	ctx.Inline = true
	ctx.inpar = true
	ctx.Process = true
	defer func() {
		bctx.loc = loc
		bctx.Macro = curMacro
		bctx.args = curArgs
		ctx.WantsSpace = ws
		ctx.Inline = false
		ctx.inpar = oldpar
		ctx.Process = proc
	}()
	bctx.loc = &location{curBlocks: blocks, curFile: loc.curFile}
	processBlocks(exp)
	if !oldpar {
		closeUnclosedBlocks(exp, "Bm")
	}
	return ctx.buf.String()
}

// reopenSpanningBlocks reopens Bm markup blocks after a paragraph break.
func reopenSpanningBlocks(exp Exporter) {
	ctx := exp.BaseContext()
	stack, ok := ctx.scopes["Bm"]
	if !ok {
		return
	}
	for _, scope := range stack {
		exp.BeginMarkupBlock(scope.tag, "")
	}
}

// closeSpanningBlocks closes Bm markup blocks at paragraph end.
func closeSpanningBlocks(exp Exporter) {
	ctx := exp.BaseContext()
	stack, ok := ctx.scopes["Bm"]
	if !ok {
		return
	}
	for i := len(stack) - 1; i >= 0; i-- {
		scope := stack[i]
		exp.EndMarkupBlock(scope.tag, scope.id, "")
	}
}

// closeUnclosedBlocks closes unclosed blocks of type given by macro, and warns
// about them.
func closeUnclosedBlocks(exp Exporter, macro string) {
	bctx := exp.BaseContext()
	if testForUnclosedBlock(exp, macro) {
		curMacro := bctx.Macro
		curArgs := bctx.args
		bctx.args = [][]ast.Inline{}
		defer func() {
			bctx.Macro = curMacro
			bctx.args = curArgs
		}()
		switch macro {
		case "Bm":
			bctx.Macro = "Em"
			for len(bctx.scopes["Bm"]) > 0 {
				s := bctx.scopes["Bm"][0]
				if s.tag != "" {
					bctx.args = append(bctx.args,
						[]ast.Inline{ast.Text("-t")}, []ast.Inline{ast.Text(s.tag)})
				}
				macroEm(exp)
			}
		case "Bl":
			bctx.Macro = "El"
			for len(bctx.scopes["Bl"]) > 0 {
				macroEl(exp)
			}
		case "Bd":
			bctx.Macro = "Ed"
			for len(bctx.scopes["Bd"]) > 0 {
				s := bctx.scopes["Bd"][0]
				if s.tag != "" {
					bctx.args = append(bctx.args,
						[]ast.Inline{ast.Text("-t")}, []ast.Inline{ast.Text(s.tag)})
				}
				macroEd(exp)
			}
		}
	}
}

// endEventualParagraph ends an eventual paragraph. The break can be ``soft''
// (for example in LaTeX this allows a list to belong to a surrounding
// paragraph).
func endEventualParagraph(exp Exporter, softbreak bool) {
	ctx := exp.Context()
	if ctx.inpar {
		parEnd(exp)
		if softbreak {
			exp.EndParagraphSoftly()
		} else {
			exp.EndParagraph()
		}
	}
}

// testForUnclosedBlock returns true if there is an unclosed block of type
// given by macro, and warns in such a case.
func testForUnclosedBlock(exp Exporter, macro string) bool {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	stack, ok := bctx.scopes[macro]
	if ok && len(stack) > 0 {
		scope := stack[len(stack)-1]
		// scope != nil
		beginmacro := scope.name
		var endmacro string
		switch beginmacro {
		case "Bm":
			endmacro = "Em"
		case "Bl":
			endmacro = "El"
		case "Bd":
			endmacro = "Ed"
		case "#if":
			endmacro = "#;"
		}
		location := bctx.scopeLocation(scope)
		var tag string
		if scope.tag != "" {
			tag = " of type " + scope.tag
		}
		var msg string
		var m = bctx.Macro
		if !ctx.Inline {
			msg = fmt.Sprintf("found %s while `.%s' macro %s%s isn't closed yet by a `.%s'",
				m, beginmacro, tag, location, endmacro)
		} else {
			var inUserMacroMsg string
			if scope.inUserMacro {
				inUserMacroMsg = " in user macro"
			}
			msg = fmt.Sprintf("unclosed inline markup block%s%s", tag, inUserMacroMsg)
		}
		bctx.Error(msg)
		return true
	}
	return false
}

// beginPhrasingMacro handles context stuff for inline macro, such as
// starting a new paragraph or adding a leading whitespace if necessary.
func beginPhrasingMacro(exp Exporter, nospace bool) {
	ctx := exp.Context()
	if ctx.inpar {
		exp.BeginPhrasingMacroInParagraph(nospace)
		return
	}
	if !ctx.Inline && !ctx.itemScope {
		exp.BeginParagraph()
		reopenSpanningBlocks(exp)
	}
	ctx.inpar = true
}

// BeginPhrasingMacroInParagraph is a function for default use with the method
// of same name of Exporter interface, which works for inline markup in most
// output formats.
func BeginPhrasingMacroInParagraph(exp Exporter, nospace bool) {
	ctx := exp.Context()
	if ctx.WantsSpace && !nospace {
		w := ctx.GetW()
		if ctx.Inline {
			fmt.Fprint(w, " ")
		} else {
			fmt.Fprint(w, "\n")
		}
	}
}

// testForUnclosedFormatBlock returns true if there is an unclosed Bf block,
// and warns about it.
func testForUnclosedFormatBlock(exp Exporter) bool {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	if ctx.bfInfo == nil {
		return false
	}
	var file string
	if bctx.loc.curFile != ctx.bfInfo.file {
		file = " of file " + ctx.bfInfo.file
	}
	var inUserMacro string
	if ctx.bfInfo.inUserMacro {
		inUserMacro = " opened inside user macro"
	}
	msg := fmt.Sprintf("`.%s' not allowed inside scope of `.Bf' macro%s at line %d%s",
		bctx.Macro, inUserMacro, ctx.bfInfo.line, file)
	bctx.Error(msg)
	return true
}

// testForUnclosedDe test for an unterminated user macro definition and warns about it.
func testForUnclosedDe(exp Exporter) {
	bctx := exp.BaseContext()
	if bctx.defInfo == nil {
		return
	}
	bctx.Error("found End Of File while `.#de' macro at line ", bctx.defInfo.line, " of file ", bctx.defInfo.file, " isn't closed by a `.#.'")
}
