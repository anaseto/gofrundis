package frundis

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"codeberg.org/anaseto/gofrundis/ast"
)

// text: as-is (not escaped), or regular (escaped + additional processing)

// -ns => no space even if wantspace

func processText(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	switch {
	case ctx.asIs:
		ctx.rawText.WriteString(ctx.InlinesToText(ctx.text))
	default:
		if !ctx.parScope {
			exp.BeginParagraph()
			ctx.parScope = true
			reopenSpanningBlocks(exp)
		} else if ctx.WantsSpace {
			// XXX: this can break tables for mom (and for markdown
			// things are not perfect either)
			fmt.Fprint(&ctx.buf, "\n")
		}
		text := exp.RenderText(ctx.text)
		if len(text) > 0 && hasBlankLine(text) {
			ctx.Error("empty line")
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

////////////// Macros /////////////////////////////////////////////

func macroBd(exp Exporter) {
	ctx := exp.Context()
	if scopeVerse(exp) {
		ctx.Error("`Bd' disallowed within verse")
		return
	}
	opts, flags, args := ctx.ParseOptions(specOptBd, ctx.Args)
	var id string
	if t, ok := opts["id"]; ok {
		id = exp.RenderText(t)
	}
	if !ctx.Process {
		if id != "" {
			ref := exp.GenRef("", id, false)
			ctx.storeID(id, IDInfo{Ref: ref, Type: BdID})
		}
		return
	}
	if containsSpace(id) {
		ctx.Error("id identifier should not contain spaces")
	}
	if len(args) > 0 {
		ctx.Error("useless arguments")
	}
	closeUnclosedScopes(exp, scopeInline)

	var tag string
	if t, ok := opts["t"]; ok {
		tag = exp.RenderText(t)
	}

	var pbreak ParagraphBreak
	if ctx.Dtags[tag].Cmd != "" {
		pbreak = ParBreakBlock
	}
	endParagraph(exp, pbreak)

	ctx.pushScope(&scope{kind: scopeBlock, macro: "Bd", tag: tag, id: id, tagRequired: flags["r"]})

	if tag != "" {
		_, ok := ctx.Dtags[tag]
		if !ok {
			ctx.Error("invalid tag:", tag)
		}
	}
	exp.BeginDisplayBlock(tag, id)
}

func macroBf(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	opts, flags, args := ctx.ParseOptions(specOptBf, ctx.Args)
	if len(args) > 0 {
		ctx.Error("useless arguments")
	}
	fmt, okFmt := opts["f"]
	bfinf := bfInfo{line: ctx.line}
	ctx.bfInfo = &bfinf
	if ctx.uMacroCall.loc != nil {
		bfinf.file = ctx.uMacroCall.loc.curFile
		bfinf.inUserMacro = true
	} else {
		bfinf.file = ctx.loc.curFile
	}
	tag, okTag := opts["t"]
	ctx.asIs = true
	if !okFmt && !okTag {
		ctx.Error("one of -f option or -t option at least required")
		bfinf.ignore = true
		return
	}
	if okTag {
		tag := ctx.InlinesToText(tag)
		bfinf.filterTag = tag
		_, okGoFilter := ctx.Filters[tag]
		if !okGoFilter {
			ctx.Error("undefined filter tag:", tag)
			bfinf.ignore = true
			return
		}
	}
	if okFmt {
		formats := strings.Split(ctx.InlinesToText(fmt), ",")
		ctx.checkFormats(formats)
		if ctx.notExportFormat(formats) {
			bfinf.ignore = true
			return
		}
	}
	if ctx.parScope {
		beginPhrasingMacro(exp, flags["ns"])
		ctx.WantsSpace = false
	}
}

func macroBl(exp Exporter) {
	ctx := exp.Context()
	if scopeVerse(exp) {
		ctx.Error("`Bl' disallowed within verse")
		return
	}
	if ctx.Process {
		macroBlProcess(exp)
	} else {
		macroBlInfos(exp)
	}
}

func macroBlInfos(exp Exporter) {
	ctx := exp.Context()
	opts, _, args := ctx.ParseOptions(specOptBl, ctx.Args)
	var tag string
	if t, ok := opts["t"]; ok {
		tag = ctx.InlinesToText(t)
	} else {
		tag = "item"
	}
	switch tag {
	case "item", "enum", "desc":
		if t, ok := opts["id"]; ok {
			id := exp.RenderText(t)
			ref := exp.GenRef("", id, false)
			ctx.storeID(id, IDInfo{Ref: ref, Type: UntitledList})
		}
	case "verse":
		ctx.Verse.Used = true
		title := processInlineMacros(exp, args)
		if title == "" {
			if t, ok := opts["id"]; ok {
				id := exp.RenderText(t)
				ref := exp.GenRef("", id, false)
				ctx.storeID(id, IDInfo{Ref: ref, Type: UntitledList})
			}
			return
		}
		ctx.Verse.verseCount++
		loXEntryInfos(exp, "lop",
			&LoXinfo{
				ID:        ctx.InlinesToText(opts["id"]),
				Title:     title,
				Count:     ctx.Verse.verseCount,
				RefPrefix: "poem"},
			strconv.Itoa(ctx.Verse.verseCount))
	case "table":
		ctx.Table.scope = true
		title := processInlineMacros(exp, args)
		if title == "" {
			if t, ok := opts["id"]; ok {
				id := exp.RenderText(t)
				ref := exp.GenRef("", id, false)
				ctx.storeID(id, IDInfo{Ref: ref, Type: UntitledList})
				ctx.Table.id = id
			}
			return
		}
		ctx.Table.title = title
		ctx.Table.TitCount++
		loXEntryInfos(exp, "lot",
			&LoXinfo{
				ID:        ctx.InlinesToText(opts["id"]),
				Title:     title,
				Count:     ctx.Table.TitCount,
				RefPrefix: "tbl"},
			strconv.Itoa(ctx.Table.TitCount))
	}
}

func macroBlProcess(exp Exporter) {
	ctx := exp.Context()
	opts, _, args := ctx.ParseOptions(specOptBl, ctx.Args)
	var tag string
	if t, ok := opts["t"]; ok {
		tag = ctx.InlinesToText(t)
	} else {
		tag = "item"
	}
	switch tag {
	case "item", "enum", "desc", "verse", "table":
		// Ok, do nothing
	default:
		ctx.Error("invalid `-t' option argument:", tag)
		tag = "item" // fallback to basic "item" list
	}
	switch tag {
	case "item", "enum", "desc":
		if len(args) > 0 {
			ctx.Error("useless arguments")
		}
	}
	closeUnclosedScopes(exp, scopeInline)
	scopes, ok := ctx.scopes[scopeBlock]
	if ok && len(scopes) > 0 {
		tag := ""
		for _, s := range scopes {
			if s.macro == "Bl" {
				tag = s.tag
			}
		}
		if tag != "" && tag != "item" && tag != "enum" {
			ctx.Error("nested list of invalid type") // TODO: really?
			return
		}
		if ctx.parScope {
			endParagraph(exp, ParBreakBlock)
		}
	} else {
		endParagraph(exp, ParBreakBlock)
	}

	ctx.pushScope(&scope{kind: scopeBlock, macro: "Bl", tag: tag})

	var id string
	if t, ok := opts["id"]; ok {
		id = exp.RenderText(t)
	}

	switch tag {
	case "verse":
		title := processInlineMacros(exp, args)
		if title != "" {
			ctx.Verse.verseCount++
			id = fmt.Sprintf("%d", ctx.Verse.verseCount)
		}
		exp.BeginVerse(title, id)
	case "desc":
		exp.BeginDescList(id)
	case "item":
		exp.BeginItemList(id)
	case "enum":
		exp.BeginEnumList(id)
	case "table":
		tableinfo := ctx.Table.info[ctx.Table.Count]
		if tableinfo.Title != "" {
			ctx.Table.TitCount++
			ctx.Table.titScope = true
		}
		exp.BeginTable(tableinfo)
	}
}

func macroBm(exp Exporter) {
	ctx := exp.Context()
	opts, flags, args := ctx.ParseOptions(specOptBm, ctx.Args)
	var id string
	if t, ok := opts["id"]; ok {
		id = exp.RenderText(t)
	}
	if !ctx.Process {
		if id != "" {
			ref := exp.GenRef("", id, false)
			ctx.storeID(id, IDInfo{Ref: ref, Type: SmID})
		}
		return
	}

	beginPhrasingMacro(exp, flags["ns"])
	ctx.WantsSpace = false
	var tag string
	if t, ok := opts["t"]; ok {
		tag = ctx.InlinesToText(t)
		_, ok := ctx.Mtags[tag]
		if !ok {
			ctx.Error("invalid tag argument to `-t' option")
		}
	}
	ctx.pushScope(&scope{kind: scopeInline, macro: "Bm", tag: tag, id: id, tagRequired: flags["r"]})
	exp.BeginMarkupBlock(tag, id)
	if len(args) > 0 {
		if !ctx.Inline {
			ctx.Error("useless arguments")
		} else {
			w := ctx.W()
			fmt.Fprint(w, renderArgs(exp, args))
		}
	}
}

func macroD(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	_, _, args := ctx.ParseOptions(specOptD, ctx.Args)
	if len(args) > 0 {
		ctx.Error("useless arguments")
	}
	if ctx.parScope {
		closeSpanningBlocks(exp)
		processParagraph(exp)
		if scopeVerse(exp) {
			exp.EndStanza()
		} else {
			exp.EndParagraph(ParBreakNormal)
		}
	}
	exp.BeginParagraph()
	ctx.parScope = true
	reopenSpanningBlocks(exp)
	exp.BeginDialogue()
	ctx.WantsSpace = false
}

func macroEd(exp Exporter) {
	ctx := exp.Context()
	if scopeVerse(exp) {
		ctx.Error("`Ed' disallowed within verse")
		return
	}
	if !ctx.Process {
		return
	}
	opts, _, args := ctx.ParseOptions(specOptEd, ctx.Args)
	if len(args) > 0 {
		ctx.Error("useless arguments")
	}
	var scope *scope
	for _, sc := range ctx.scopes[scopeBlock] {
		if sc.macro == "Bd" {
			scope = sc
		}
	}
	if scope == nil {
		ctx.Error("no corresponding `.Bd'")
		return
	}
	if tag, ok := opts["t"]; ok {
		if ctx.InlinesToText(tag) != scope.tag {
			location := ctx.scopeLocation(scope)
			ctx.Errorf("tag doesn't match tag '%s' of current block opened %s", scope.tag, location)
		}
	} else if scope.tagRequired {
		location := ctx.scopeLocation(scope)
		ctx.Errorf("missing required tag matching tag '%s' of current block opened %s", scope.tag, location)
	}
	var pbreak ParagraphBreak
	if ctx.Dtags[scope.tag].Cmd != "" {
		pbreak = ParBreakBlock
	}
	closeUnclosedScopes(exp, scopeInline)
	closeUnclosedBlocks(exp, "Bd")
	ctx.popScope(scopeBlock)
	endParagraph(exp, pbreak)
	exp.EndDisplayBlock(scope.tag)

	ctx.WantsSpace = false
}

func macroEf(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	_, flags, args := ctx.ParseOptions(specOptEf, ctx.Args)
	if len(args) > 0 {
		ctx.Error("useless arguments")
	}
	if ctx.bfInfo == nil {
		ctx.Error("no corresponding `.Bf'")
		return
	}
	if !ctx.bfInfo.ignore {
		var text string
		if tag := ctx.bfInfo.filterTag; tag != "" {
			filter, ok := ctx.Filters[tag]
			if ok {
				text = filter(ctx.rawText.String())
			} else {
				ctx.Error("invalid filter tag:", tag)
				text = ctx.rawText.String()
			}
		} else {
			text = ctx.rawText.String()
		}
		w := ctx.W()
		fmt.Fprint(w, text)
		if ctx.parScope && !flags["ns"] {
			ctx.WantsSpace = true
		} else if !flags["ns"] {
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
	if ctx.Table.scope {
		if ctx.Table.cols == 0 {
			ctx.Table.cols = ctx.Table.Cell
		}
		ctx.Table.info = append(ctx.Table.info,
			&TableData{
				Cols:  ctx.Table.cols,
				ID:    ctx.Table.id,
				Title: ctx.Table.title})
		ctx.Table.Count++
		ctx.Table = TableInfo{
			info:     ctx.Table.info,
			Count:    ctx.Table.Count,
			TitCount: ctx.Table.TitCount}
	}
}

func macroElProcess(exp Exporter) {
	ctx := exp.Context()
	var scope *scope
	for _, s := range ctx.scopes[scopeBlock] {
		if s.macro == "Bl" {
			scope = s
		}
	}
	if scope == nil {
		ctx.Error("no corresponding `.Bl'")
		return
	}
	closeUnclosedScopes(exp, scopeInline)
	closeUnclosedBlocks(exp, "It")
	itemScope := false
	for sc := ctx.popScope(scopeBlock); sc.macro != "Bl"; sc = ctx.popScope(scopeBlock) {
		if sc.macro == "It" {
			itemScope = true
		}
	}
	_, _, args := ctx.ParseOptions(specOptEl, ctx.Args)
	if len(args) > 0 {
		ctx.Error("useless arguments")
	}
	if !itemScope {
		switch scope.tag {
		case "desc":
			ctx.Error("no previous `.It' in 'desc' list. Empty list?")
			exp.BeginDescValue()
		case "item":
			ctx.Error("no previous `.It'. Empty list?")
			exp.BeginItem()
		case "enum":
			ctx.Error("no previous `.It'. Empty list?")
			exp.BeginEnumItem()
		case "verse":
			if ctx.parScope && !ctx.verseScope {
				ctx.Error("unexpected accumulated text outside item scope")
			}
		default:
			if ctx.parScope {
				ctx.Error("unexpected accumulated text outside item scope")
			}
		}
	}

	switch scope.tag {
	case "verse":
		processParagraph(exp)
		closeUnclosedScopes(exp, scopeInline)
		ctx.verseScope = false
		exp.EndStanza()
		exp.EndVerse()
	case "desc":
		endParagraph(exp, ParBreakItem)
		exp.EndDescValue()
		exp.EndDescList()
	case "enum":
		endParagraph(exp, ParBreakItem)
		closeUnclosedScopes(exp, scopeInline)
		exp.EndEnumItem()
		exp.EndEnumList()
	case "item":
		endParagraph(exp, ParBreakItem)
		closeUnclosedScopes(exp, scopeInline)
		exp.EndItem()
		exp.EndItemList()
	case "table":
		// allow empty table
		if itemScope {
			endParagraph(exp, ParBreakItem)
			closeUnclosedScopes(exp, scopeInline)
			exp.EndTableCell()
			exp.EndTableRow()
		}
		exp.EndTable(ctx.Table.info[ctx.Table.Count])
		ctx.Table.titScope = false
		ctx.Table.scope = false
		ctx.Table.Cell = 0
		ctx.Table.cols = 0
		ctx.Table.Count++
	}
	ctx.WantsSpace = false
}

func macroEm(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	opts, flags, args := ctx.ParseOptions(specOptEm, ctx.Args)
	scope := ctx.popScope(scopeInline)
	if scope == nil {
		ctx.Error("no corresponding `.Bm'")
		return
	}
	if tag, ok := opts["t"]; ok {
		if ctx.InlinesToText(tag) != scope.tag {
			location := ctx.scopeLocation(scope)
			ctx.Errorf("tag doesn't match tag '%s' of current block opened %s", scope.tag, location)
		}
	} else if scope.tagRequired {
		location := ctx.scopeLocation(scope)
		ctx.Errorf("missing required tag matching tag '%s' of current block opened %s", scope.tag, location)
	}
	tag := scope.tag
	id := scope.id
	var punct string
	if len(args) > 0 {
		if !ctx.Inline || ctx.isPunctArg(args[0]) {
			punct = exp.RenderText(args[0])
			args = args[1:]
		}
	}
	exp.EndMarkupBlock(tag, id, punct)
	if len(args) > 0 {
		if !ctx.Inline {
			ctx.Error("useless args in macro `.Em'")
		} else {
			w := ctx.W()
			fmt.Fprint(w, renderArgs(exp, args))
		}
	}
	ctx.WantsSpace = !flags["ns"]
}

func macroFt(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	opts, flags, args := ctx.ParseOptions(specOptFt, ctx.Args)
	format, okFmt := opts["f"]
	if okFmt {
		formats := strings.Split(ctx.InlinesToText(format), ",")
		ctx.checkFormats(formats)
		if ctx.notExportFormat(formats) {
			return
		}
	}
	tag, okTag := opts["t"]
	if !okFmt && !okTag {
		ctx.Error("one of -f option or -t option at least required")
		return
	}
	if ctx.parScope {
		beginPhrasingMacro(exp, flags["ns"])
		ctx.WantsSpace = false
	}
	var text string
	if okTag {
		tag := ctx.InlinesToText(tag)
		goFilter, okGoFilter := ctx.Filters[tag]
		if okGoFilter {
			text = goFilter(argsToText(exp, args))
		} else {
			ctx.Error("undefined filter tag:", tag)
			text = renderArgs(exp, args)
		}
	} else {
		text = argsToText(exp, args)
	}
	w := ctx.W()
	fmt.Fprint(w, text)
}

func macroIncludeFile(exp Exporter) {
	ctx := exp.Context()
	opts, flags, args := ctx.ParseOptions(specOptIncludeFile, ctx.Args)
	if format, ok := opts["f"]; ok {
		formats := strings.Split(ctx.InlinesToText(format), ",")
		if ctx.Process {
			ctx.checkFormats(formats)
		}
		if ctx.notExportFormat(formats) {
			return
		}
	}
	if len(args) == 0 {
		if ctx.Process {
			ctx.Error("filename argument required")
		}
		return
	}
	filename := ctx.InlinesToText(args[0])
	if flags["as-is"] {
		if !ctx.Process {
			return
		}
		if ctx.parScope {
			beginPhrasingMacro(exp, flags["ns"])
			ctx.WantsSpace = true
		}
		source, err := os.ReadFile(filename)
		if err != nil {
			ctx.Error("as-is inclusion:", err)
			return
		}
		var text string
		if t, ok := opts["t"]; ok {
			tag := ctx.InlinesToText(t)
			if filter, ok := ctx.Filters[tag]; ok {
				text = filter(string(source))
			} else {
				text = string(source)
				ctx.Error("unknown tag:", tag)
			}
		} else {
			text = string(source)
		}
		w := ctx.W()
		fmt.Fprint(w, text)
	} else {
		// frundis source file
		filename, ok := SearchIncFile(exp, filename)
		if !ok {
			if ctx.Process {
				ctx.Errorf("%s: no such frundis source file", filename)
			}
			return
		}
		err := processFile(exp, filename)
		if err != nil {
			ctx.Error(err)
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
	ctx := exp.Context()
	opts, flags, args := ctx.ParseOptions(specOptIm, ctx.Args)
	if len(args) == 0 {
		ctx.Error("arguments required")
		return
	}
	var punct string
	if len(args) > 1 {
		args, punct = getClosePunct(exp, args)
	}
	var link string
	if t, ok := opts["link"]; ok {
		link = ctx.InlinesToText(t)
	}
	var alt string
	if t, ok := opts["alt"]; ok {
		alt = ctx.InlinesToText(t)
	}
	if len(args) > 2 {
		ctx.Error("too many arguments")
		args = args[:2]
	}
	switch len(args) {
	case 0:
		ctx.Error("requires at least one argument")
	case 1:
		beginPhrasingMacro(exp, flags["ns"])
		ctx.WantsSpace = true
		image := ctx.InlinesToText(args[0])
		var id string
		if t, ok := opts["id"]; ok {
			id = exp.RenderText(t)
		}
		exp.InlineImage(image, link, id, punct, alt)
	case 2:
		closeUnclosedScopes(exp, scopeInline)
		endParagraph(exp, ParBreakNormal)
		image := ctx.InlinesToText(args[0])
		caption := exp.RenderText(args[1])
		ctx.FigCount++
		exp.FigureImage(image, caption, link, alt)
	}
}

func macroImInfos(exp Exporter) {
	ctx := exp.Context()
	opts, _, args := ctx.ParseOptions(specOptIm, ctx.Args)
	if len(args) == 0 {
		return
	}
	if len(args) > 1 {
		args, _ = getClosePunct(exp, args)
	}
	var image string
	if len(args) == 0 {
		return
	}
	if len(args) == 1 {
		// inline image
		image = ctx.InlinesToText(args[0])
		ctx.Images = append(ctx.Images, image)
		if t, ok := opts["id"]; ok {
			id := exp.RenderText(t)
			ref := exp.GenRef("", id, false)
			ctx.storeID(id, IDInfo{Ref: ref, Type: InlineImID})
		}
		return
	}
	// figure
	image = ctx.InlinesToText(args[0])
	ctx.Images = append(ctx.Images, image)
	label := exp.RenderText(args[1])
	ctx.FigCount++
	loXEntryInfos(exp, "lof",
		&LoXinfo{
			ID:        ctx.InlinesToText(opts["id"]),
			Title:     label,
			Count:     ctx.FigCount,
			RefPrefix: "fig"},
		strconv.Itoa(ctx.FigCount))
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
	if ctx.Table.scope {
		if ctx.Table.cols == 0 {
			ctx.Table.cols = ctx.Table.Cell
		}
		ctx.Table.Cell = 1
	}
}

func macroItProcess(exp Exporter) {
	ctx := exp.Context()
	_, _, args := ctx.ParseOptions(specOptIt, ctx.Args)
	scopes := ctx.scopes[scopeBlock]
	var s *scope
	for _, sc := range scopes {
		if sc.macro == "Bl" {
			s = sc
		}
	}
	if s == nil {
		ctx.Error("outside `.Bl' macro scope")
		return
	}
	ctx.WantsSpace = false
	switch s.tag {
	case "desc":
		closeUnclosedScopes(exp, scopeInline)
		closeUnclosedBlocks(exp, "It")
		macroItDesc(exp, args)
	case "item", "enum":
		closeUnclosedScopes(exp, scopeInline)
		closeUnclosedBlocks(exp, "It")
		macroItemenum(exp, args, s.tag)
	case "table":
		closeUnclosedScopes(exp, scopeInline)
		closeUnclosedBlocks(exp, "It")
		macroItTable(exp, args)
	case "verse":
		macroItVerse(exp, args)
	}
}

func macroItDesc(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	closeUnclosedBlocks(exp, "It")
	if scopeIt(exp) {
		endParagraph(exp, ParBreakItem)
		exp.EndDescValue()
	} else if ctx.parScope {
		ctx.Error("previous text outside of It scope")
	}
	if len(args) == 0 {
		ctx.Error("description name required")
	}
	name := processInlineMacros(exp, args)
	ctx.WantsSpace = false
	exp.DescName(name)
	exp.BeginDescValue()
	ctx.parScope = false
	if !scopeIt(exp) {
		ctx.pushScope(&scope{kind: scopeBlock, macro: "It"})
	}
}

func macroItemenum(exp Exporter, args [][]ast.Inline, tag string) {
	ctx := exp.Context()
	closeUnclosedBlocks(exp, "It")
	if scopeIt(exp) {
		endParagraph(exp, ParBreakItem)
		switch tag {
		case "item":
			exp.EndItem()
		case "enum":
			exp.EndEnumItem()
		}
	} else if ctx.parScope {
		ctx.Error("previous text outside of It scope")
	}
	switch tag {
	case "item":
		exp.BeginItem()
	case "enum":
		exp.BeginEnumItem()
	}
	ctx.parScope = false
	ctx.WantsSpace = false
	if len(args) > 0 {
		exp.BeginParagraph()
		ctx.parScope = true
		w := ctx.W()
		fmt.Fprint(w, processInlineMacros(exp, args))
		ctx.WantsSpace = true
	}
	if !scopeIt(exp) {
		ctx.pushScope(&scope{kind: scopeBlock, macro: "It"})
	}
}

func macroItTable(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	closeUnclosedBlocks(exp, "It")
	if scopeIt(exp) {
		endParagraph(exp, ParBreakItem)
		exp.EndTableCell()
		exp.EndTableRow()
	} else if ctx.parScope {
		ctx.Error("previous text outside of It scope")
	}
	if ctx.Table.cols == 0 {
		ctx.Table.cols = ctx.Table.Cell
	}
	if ctx.Table.cols > ctx.Table.Cell {
		ctx.Error("not enough cells in previous row")
	}
	ctx.Table.Cell = 1
	exp.BeginTableRow()
	exp.BeginTableCell()
	ctx.parScope = false
	if len(args) > 0 {
		w := ctx.W()
		fmt.Fprint(w, processInlineMacros(exp, args))
		ctx.WantsSpace = true
	}
	if !scopeIt(exp) {
		ctx.pushScope(&scope{kind: scopeBlock, macro: "It"})
	}
}

func macroItVerse(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	if !ctx.parScope {
		exp.BeginParagraph()
		exp.BeginVerseLine()
		ctx.parScope = true
	} else {
		if !ctx.verseScope {
			ctx.Error("found verse text outside of It scope")
		} else {
			exp.EndVerseLine()
		}
		exp.BeginVerseLine()
	}
	if len(args) > 0 {
		w := ctx.W()
		fmt.Fprint(w, processInlineMacros(exp, args))
		ctx.WantsSpace = true
	}
	ctx.verseScope = true
}

func macroLk(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	_, flags, args := ctx.ParseOptions(specOptLk, ctx.Args)
	var punct string
	if len(args) > 1 {
		args, punct = getClosePunct(exp, args)
	}
	if len(args) == 0 {
		ctx.Error("argument required")
		return
	}
	beginPhrasingMacro(exp, flags["ns"])
	ctx.WantsSpace = true

	if len(args) >= 2 {
		url := ctx.InlinesToText(args[0])
		label := processInlineMacros(exp, args[1:])
		exp.LkWithLabel(url, label, punct)
	} else {
		url := ctx.InlinesToText(args[0])
		exp.LkWithoutLabel(url, punct)
	}
}

func macroP(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	_, _, args := ctx.ParseOptions(specOptP, ctx.Args)
	if ctx.parScope {
		closeSpanningBlocks(exp)
		processParagraph(exp)
		if scopeVerse(exp) {
			exp.EndStanza()
		} else {
			exp.EndParagraph(ParBreakNormal)
		}
	} else {
		exp.EndParagraph(ParBreakForced)
		ctx.parScope = false
	}
	if len(args) > 0 {
		ctx.parScope = true
		title := processInlineMacros(exp, args)
		exp.ParagraphTitle(title)
		reopenSpanningBlocks(exp)
	}
	ctx.WantsSpace = false
	ctx.verseScope = false
}

func macroSm(exp Exporter) {
	ctx := exp.Context()
	opts, flags, args := ctx.ParseOptions(specOptSm, ctx.Args)
	var id string
	if t, ok := opts["id"]; ok {
		id = exp.RenderText(t)
	}
	if !ctx.Process {
		if id != "" {
			ref := exp.GenRef("", id, false)
			ctx.storeID(id, IDInfo{Ref: ref, Type: SmID})
		}
		return
	}
	if len(args) == 0 {
		ctx.Error("arguments required")
		return
	}
	var punct string
	if len(args) > 1 {
		args, punct = getClosePunct(exp, args)
	}

	beginPhrasingMacro(exp, flags["ns"])
	var tag string
	if t, ok := opts["t"]; ok {
		tag = ctx.InlinesToText(t)
		_, ok := ctx.Mtags[tag]
		if !ok {
			ctx.Error("invalid tag argument to `-t' option")
		}
	}
	exp.BeginMarkupBlock(tag, id)
	w := ctx.W()
	fmt.Fprint(w, renderArgs(exp, args))
	exp.EndMarkupBlock(tag, id, punct)
	ctx.WantsSpace = true
}

func macroSx(exp Exporter) {
	ctx := exp.Context()
	if !ctx.Process {
		return
	}
	_, flags, args := ctx.ParseOptions(specOptSx, ctx.Args)
	var punct string
	if len(args) > 1 {
		args, punct = getClosePunct(exp, args)
	}
	if len(args) == 0 {
		ctx.Error("arguments required")
		return
	}
	id := ctx.InlinesToText(args[0])
	idinfo, ok := ctx.IDs[id]
	if !ok {
		ctx.Error("reference to unknown id:", id)
	}
	beginPhrasingMacro(exp, flags["ns"])
	ctx.WantsSpace = true
	if len(args) > 1 {
		args = args[1:]
		idinfo.Name = processInlineMacros(exp, args)
	} else if idinfo.Name == "" {
		idinfo.Name = id
	}
	exp.CrossReference(idinfo, punct)
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
	ctx.Table.Cell++
}

func macroTaProcess(exp Exporter) {
	ctx := exp.Context()
	_, _, args := ctx.ParseOptions(specOptTa, ctx.Args)
	scopes := ctx.scopes[scopeBlock]
	inTableScope := false
	for _, s := range scopes {
		if s.macro == "Bl" {
			if s.tag == "table" {
				inTableScope = true
			} else {
				inTableScope = false
			}
		}
	}
	if !inTableScope {
		ctx.Error("outside `.Bl -t table' macro scope")
		return
	}
	closeUnclosedScopes(exp, scopeInline)
	closeUnclosedBlocks(exp, "It")
	scopes = ctx.scopes[scopeBlock]
	var s *scope
	if scopeIt(exp) {
		s = scopes[len(scopes)-2]
	} else {
		ctx.Error("there are no rows")
		return
	}
	if s.tag != "table" {
		ctx.Error("not a ``table'' list", s.tag, s.macro)
		return
	}
	if !scopeIt(exp) {
		ctx.Error("outside an `.It' row scope")
		return
	}
	closeUnclosedScopes(exp, scopeInline)
	endParagraph(exp, ParBreakItem)
	exp.EndTableCell()
	ctx.Table.Cell++
	exp.BeginTableCell()
	ctx.parScope = false
	if len(args) > 0 {
		exp.BeginParagraph()
		ctx.parScope = true
		w := ctx.W()
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
	ctx := exp.Context()
	_, flags, _ := ctx.ParseOptions(specOptTc, ctx.Args)
	exp.TableOfContentsInfos(flags)
}

func macroTcProcess(exp Exporter) {
	ctx := exp.Context()
	closeUnclosedScopes(exp, scopeInline)
	switch ctx.Format {
	case "xhtml", "epub":
		// allow blocks when exporting to xhtml or epub
	default:
		closeUnclosedScopes(exp, scopeBlock)
	}
	opts, flags, args := ctx.ParseOptions(specOptTc, ctx.Args)
	if len(args) > 0 {
		ctx.Error("useless arguments")
	}
	var pbreak ParagraphBreak
	if flags["ns"] {
		pbreak = ParBreakBlock
	}
	endParagraph(exp, pbreak)
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
		ctx.Error("only one of the -toc, -lof and -lot options should bet set")
		return
	}
	exp.TableOfContents(opts, flags)
}

func macroX(exp Exporter) {
	ctx := exp.Context()
	if ctx.Process {
		return
	}
	args := ctx.Args
	if len(args) == 0 {
		ctx.Error("not enough arguments")
		return
	}
	cmd := ctx.InlinesToText(args[0])
	args = args[1:]
	ctx.Macro += " " + cmd
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
	var opts map[string][]ast.Inline
	opts, _, _ = ctx.ParseOptions(specOptXdtag, args)
	var formats []string
	if format, okFmt := opts["f"]; okFmt {
		formats = strings.Split(ctx.InlinesToText(format), ",")
	} else {
		ctx.Error("`-f' option required")
		return
	}
	ctx.checkFormats(formats)
	if ctx.notExportFormat(formats) {
		return
	}
	var tag string
	if t, ok := opts["t"]; ok {
		tag = ctx.InlinesToText(t)
		if tag == "" {
			ctx.Error("tag option argument cannot be empty")
			return
		}
	} else {
		ctx.Error("-t option required")
		return
	}
	var pairs []string
	if t, ok := opts["a"]; ok {
		s := ctx.InlinesToText(t)
		var err error
		pairs, err = readPairs(s)
		if err != nil {
			ctx.Error("invalid -a argument (missing separator?):", err)
		}
		checkPairs(ctx, pairs)
	}
	cmd := ctx.InlinesToText(opts["c"])
	ctx.Dtags[tag] = exp.Xdtag(cmd, pairs)
}

func macroXftag(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	opts, flags, args := ctx.ParseOptions(specOptXftag, args)
	if format, ok := opts["f"]; ok {
		formats := strings.Split(ctx.InlinesToText(format), ",")
		ctx.checkFormats(formats)
		if len(formats) > 0 && ctx.notExportFormat(formats) {
			return
		}
	}
	var tag string
	if t, ok := opts["t"]; ok {
		tag = ctx.InlinesToText(t)
		if tag == "" {
			ctx.Error("tag option argument cannot be empty")
			return
		}
	} else {
		ctx.Error("-t option should be specified")
		return
	}
	if flags["shell"] {
		if len(args) <= 0 {
			ctx.Error("missing arguments for shell command")
			return
		}
		sargs := make([]string, 0, len(args))
		for _, elt := range args {
			sargs = append(sargs, ctx.InlinesToText(elt))
		}
		ctx.Filters[tag] = func(text string) string { return shellFilter(exp, sargs, text) }
		return
	}
	if t, ok := opts["gsub"]; ok {
		s := ctx.InlinesToText(t)
		sr := strings.NewReader(s)
		r, size, err := sr.ReadRune()
		if err != nil {
			ctx.Error("invalid -gsub argument")
			return
		}
		s = s[size:]
		repls := strings.Split(s, string(r))
		if len(repls)%2 != 0 {
			ctx.Error("invalid -gsub argument (non even number of strings)")
			return
		}
		escaper := strings.NewReplacer(repls...)
		ctx.Filters[tag] = func(text string) string { return escaper.Replace(text) }
		return
	}
	if t, ok := opts["regexp"]; ok {
		s := ctx.InlinesToText(t)
		sr := strings.NewReader(s)
		r, size, err := sr.ReadRune()
		if err != nil {
			ctx.Error("invalid -regexp argument")
			return
		}
		s = s[size:]
		repls := strings.Split(s, string(r))
		if len(repls) != 2 {
			ctx.Error("invalid -regexp argument (missing separator?)")
			return
		}
		rx, err := regexp.Compile(repls[0])
		if err != nil {
			ctx.Error("invalid -regexp argument:", err)
			return
		}
		ctx.Filters[tag] = func(text string) string { return rx.ReplaceAllString(text, repls[1]) }
		return
	}

	ctx.Error("one of -shell/-gsub/-regexp option should be provided")
}

func macroXmtag(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	var opts map[string][]ast.Inline
	opts, _, _ = ctx.ParseOptions(specOptXmtag, args)
	var formats []string
	if format, okFmt := opts["f"]; okFmt {
		formats = strings.Split(ctx.InlinesToText(format), ",")
	} else {
		ctx.Error("`-f' option required")
		return
	}
	ctx.checkFormats(formats)
	if ctx.notExportFormat(formats) {
		return
	}
	var tag string
	if t, ok := opts["t"]; ok {
		tag = ctx.InlinesToText(t)
		if tag == "" {
			ctx.Error("tag option argument cannot be empty")
			return
		}
	} else {
		ctx.Error("-t option should be specified")
		return
	}
	b, e := exp.RenderText(opts["b"]), exp.RenderText(opts["e"])
	var cmd *string
	if t, ok := opts["c"]; ok {
		s := ctx.InlinesToText(t)
		cmd = &s
	}
	var pairs []string
	if t, ok := opts["a"]; ok {
		s := ctx.InlinesToText(t)
		var err error
		pairs, err = readPairs(s)
		if err != nil {
			ctx.Error("invalid -a argument (missing separator?):", err)
		}
		checkPairs(ctx, pairs)
	}
	ctx.Mtags[tag] = exp.Xmtag(cmd, b, e, pairs)
}

func macroXset(exp Exporter, args [][]ast.Inline) {
	ctx := exp.Context()
	var opts map[string][]ast.Inline
	opts, _, args = ctx.ParseOptions(specOptXset, args)
	var formats []string
	if format, okFmt := opts["f"]; okFmt {
		formats = strings.Split(ctx.InlinesToText(format), ",")
	}
	ctx.checkFormats(formats)
	if len(formats) > 0 && ctx.notExportFormat(formats) {
		return
	}
	if len(args) < 2 {
		ctx.Error("two arguments expected")
		return
	}
	if len(args) > 2 {
		ctx.Error("too many arguments")
	}
	param := ctx.InlinesToText(args[0])
	var value string
	switch param {
	case "dmark", "document-author", "document-date", "document-title",
		"epub-cover", "epub-css", "epub-metadata", "epub-subject", "epub-uuid", "epub-version", "epub-nav-landmarks",
		"lang",
		"latex-preamble", "latex-variant",
		"mom-preamble",
		"nbsp", "title-page",
		"xhtml-bottom", "xhtml-css", "xhtml-index", "xhtml-favicon", "xhtml-go-up", "xhtml-top", "xhtml-version", "xhtml-chap-prefix", "xhtml-chap-custom-filenames", "xhtml-custom-ids":
	default:
		ctx.Error("unknown parameter:", param)
	}
	switch param {
	case "document-author", "document-date", "document-title",
		"epub-subject", "epub-uuid",
		"xhtml-index", "xhtml-go-up", "xhtml-top":
		value = exp.RenderText(args[1])
	default:
		value = ctx.InlinesToText(args[1])
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
	opts, flags, args := ctx.ParseOptions(specOptHeader, ctx.Args)
	if len(args) == 0 {
		ctx.Error("arguments required")
		return
	}
	numbered := !flags["nonum"]
	closeUnclosedScopes(exp, scopeInline)
	closeUnclosedScopes(exp, scopeBlock)
	endParagraph(exp, ParBreakNormal)
	ctx.Toc.updateHeadersCount(ctx.Macro, flags["nonum"])
	title := processInlineMacros(exp, args)
	ctx.IDX = ctx.InlinesToText(opts["id"])
	if ctx.Macro == "Ch" || ctx.Macro == "Pt" {
		ctx.ID = ctx.IDX
	}
	exp.BeginHeader(ctx.Macro, numbered, title)
	fmt.Fprint(ctx.W(), title)
	closeUnclosedScopes(exp, scopeInline)
	exp.EndHeader(ctx.Macro, numbered, title)
}

func macroHeaderInfos(exp Exporter) {
	ctx := exp.Context()
	opts, flags, args := ctx.ParseOptions(specOptHeader, ctx.Args)
	if len(args) == 0 {
		// Error message while processing
		return
	}
	ctx.Toc.updateHeadersCount(ctx.Macro, flags["nonum"])
	switch ctx.Macro {
	case "Pt":
		ctx.Toc.HasPart = true
	case "Ch":
		ctx.Toc.HasChapter = true
	}
	id := ctx.InlinesToText(opts["id"])
	ctx.IDX = id
	if ctx.Macro == "Ch" || ctx.Macro == "Pt" {
		ctx.ID = id
	}
	ref := exp.HeaderReference(ctx.Macro)
	num := ctx.Toc.HeaderNum(ctx.Macro, flags["nonum"])
	if id != "" {
		ctx.storeID(id, IDInfo{Ref: ref, Name: num, Type: HeaderID})
	}
	title := processInlineMacros(exp, args)
	info := &LoXinfo{
		Count:     ctx.Toc.HeaderCount,
		ID:        id,
		Ref:       ref,
		RefPrefix: "s",
		Macro:     ctx.Macro,
		Nonum:     flags["nonum"],
		Title:     title,
		Num:       num}
	ctx.LoXstack["toc"] = append(ctx.LoXstack["toc"], info)
	switch ctx.Macro {
	case "Pt", "Ch":
		ctx.LoXstack["nav"] = append(ctx.LoXstack["nav"], info)
	}
}

////////////// Macro utilities ///////////////////////////////////////////

// processInlineMacros processes a list of arguments with Sm-like markup and
// returns the result.
func processInlineMacros(exp Exporter, args [][]ast.Inline) string {
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
						Line: ctx.line,
						Name: ctx.inlineToText(arg[0])})
				break
			}
			fallthrough
		default:
			if len(blocks) == 0 {
				// No Sm or Bm as first argument: initialize text block
				blocks = append(blocks, &ast.TextBlock{Line: ctx.line})
			}
			b := blocks[len(blocks)-1]
			switch b := b.(type) {
			case *ast.Macro:
				b.Args = append(b.Args, arg)
			case *ast.TextBlock:
				if len(b.Text) > 0 {
					b.Text = append(b.Text, ast.Text(" "))
				}
				b.Text = append(b.Text, arg...)
			}
		}
	}
	if len(blocks) == 0 {
		// Ensure that we have at least one block, allowing for line
		// error reporting should an error occur.
		blocks = append(blocks, &ast.TextBlock{Line: ctx.line})
	}
	loc := ctx.loc
	curMacro := ctx.Macro
	curArgs := ctx.Args
	ws := ctx.WantsSpace
	oldpar := ctx.parScope
	proc := ctx.Process
	if !ctx.Process {
		ctx.quiet = true
	}
	ctx.WantsSpace = false
	ctx.Inline = true
	ctx.parScope = true
	ctx.Process = true
	defer func() {
		ctx.loc = loc
		ctx.Macro = curMacro
		ctx.Args = curArgs
		ctx.WantsSpace = ws
		ctx.Inline = false
		ctx.parScope = oldpar
		ctx.Process = proc
		ctx.quiet = false
	}()
	ctx.loc = &location{curBlocks: blocks, curFile: loc.curFile}
	processBlocks(exp)
	if !oldpar {
		closeUnclosedScopes(exp, scopeInline)
	}
	return ctx.buf.String()
}

// beginPhrasingMacro handles context stuff for inline macro, such as
// starting a new paragraph or adding a leading whitespace if necessary.
func beginPhrasingMacro(exp Exporter, nospace bool) {
	ctx := exp.Context()
	if ctx.parScope {
		exp.BeginPhrasingMacroInParagraph(nospace)
		return
	}
	if !ctx.Inline && !scopeVerse(exp) {
		exp.BeginParagraph()
		reopenSpanningBlocks(exp)
	}
	ctx.parScope = true
}

// BeginPhrasingMacroInParagraph is a function for default use with the method
// of same name of Exporter interface, which works for inline markup in most
// output formats.
func BeginPhrasingMacroInParagraph(exp Exporter, nospace bool) {
	ctx := exp.Context()
	if ctx.WantsSpace && !nospace {
		w := ctx.W()
		if ctx.Inline {
			fmt.Fprint(w, " ")
		} else {
			fmt.Fprint(w, "\n")
		}
	}
}

// getClosePunct returns a punctuation delimiter or an empty string, and an
// updated arguments slice.
func getClosePunct(exp Exporter, args [][]ast.Inline) ([][]ast.Inline, string) {
	ctx := exp.Context()
	last := args[len(args)-1]
	hasPunct := false
	if ctx.isPunctArg(last) {
		hasPunct = true
		args = args[:len(args)-1]
	}
	var punct string
	if hasPunct {
		punct = exp.RenderText(last)
	}
	return args, punct
}

// reopenSpanningBlocks reopens Bm markup blocks after a paragraph break.
func reopenSpanningBlocks(exp Exporter) {
	ctx := exp.Context()
	stack, ok := ctx.scopes[scopeInline]
	if !ok {
		return
	}
	for _, scope := range stack {
		exp.BeginMarkupBlock(scope.tag, "")
	}
}

// closeSpanningBlocks closes Bm markup blocks at paragraph end.
func closeSpanningBlocks(exp Exporter) {
	ctx := exp.Context()
	stack, ok := ctx.scopes[scopeInline]
	if !ok {
		return
	}
	for i := len(stack) - 1; i >= 0; i-- {
		scope := stack[i]
		exp.EndMarkupBlock(scope.tag, scope.id, "")
	}
}

// closeUnclosedScopes closes unclosed scopes of a given type.
func closeUnclosedScopes(exp Exporter, scope scopeKind) {
	ctx := exp.Context()
	if len(ctx.scopes[scope]) > 0 {
		curMacro := ctx.Macro
		curArgs := ctx.Args
		ctx.Args = [][]ast.Inline{}
		defer func() {
			ctx.Macro = curMacro
			ctx.Args = curArgs
		}()
		switch scope {
		case scopeInline:
			for len(ctx.scopes[scopeInline]) > 0 {
				scopes := ctx.scopes[scopeInline]
				ctx.Macro = curMacro
				s := scopes[len(scopes)-1]
				warnUnclosedScope(exp, s)
				ctx.Macro = "Em"
				if s.tag != "" {
					ctx.Args = append(ctx.Args,
						[]ast.Inline{ast.Text("-t")}, []ast.Inline{ast.Text(s.tag)})
				}
				oquiet := ctx.quiet
				ctx.quiet = true
				macroEm(exp)
				ctx.quiet = oquiet
				ctx.Args = ctx.Args[:0]
			}
		case scopeBlock:
			for len(ctx.scopes[scopeBlock]) > 0 {
				scopes := ctx.scopes[scopeBlock]
				s := scopes[len(scopes)-1]
				ctx.Macro = curMacro
				warnUnclosedScope(exp, s)
				switch s.macro {
				case "Bl", "It":
					ctx.Macro = "El"
				default:
					ctx.Macro = "Ed"
				}
				if s.tag != "" {
					ctx.Args = append(ctx.Args,
						[]ast.Inline{ast.Text("-t")}, []ast.Inline{ast.Text(s.tag)})
				}
				oquiet := ctx.quiet
				ctx.quiet = true
				switch s.macro {
				case "Bl", "It":
					macroEl(exp)
				default:
					macroEd(exp)
				}
				ctx.quiet = oquiet
				ctx.Args = ctx.Args[:0]
			}
		}
	}
}

// closeUnclosedBlocks closes unclosed blocks of a different macro than the
// given one.
func closeUnclosedBlocks(exp Exporter, macro string) {
	ctx := exp.Context()
	scopes := ctx.scopes[scopeBlock]
	if len(scopes) <= 1 {
		return
	}
	inScope := false
	for _, s := range scopes {
		if s.macro == macro {
			inScope = true
			break
		}
	}
	if !inScope {
		return
	}
	s := scopes[len(scopes)-1]
	if s.macro == ctx.Macro || macro == "It" && s.macro == "Bl" {
		return
	}
	curMacro := ctx.Macro
	curArgs := ctx.Args
	ctx.Args = [][]ast.Inline{}
	defer func() {
		ctx.Macro = curMacro
		ctx.Args = curArgs
	}()
	for i := len(scopes) - 1; i >= 0; i-- {
		s := scopes[i]
		if s.macro == macro || macro == "It" && s.macro == "Bl" {
			return
		}
		endParagraph(exp, ParBreakNormal)
		location := ctx.scopeLocation(s)
		var tag string
		if s.tag != "" {
			tag = " of type " + s.tag
		}
		switch s.macro {
		case "Bl", "It":
			msg := fmt.Sprintf("found %s while `.%s' macro%s %s isn't closed yet by a `.%s'",
				curMacro, s.macro, tag, location, "El")
			ctx.Error(msg)
			ctx.Macro = "El"
		default:
			msg := fmt.Sprintf("found %s while `.%s' macro%s %s isn't closed yet by a `.%s'",
				curMacro, s.macro, tag, location, "Ed")
			ctx.Error(msg)
			ctx.Macro = "Ed"
		}
		if s.tag != "" {
			ctx.Args = append(ctx.Args,
				[]ast.Inline{ast.Text("-t")}, []ast.Inline{ast.Text(s.tag)})
		}
		oquiet := ctx.quiet
		ctx.quiet = true
		switch s.macro {
		case "Bl", "It":
			macroEl(exp)
		default:
			macroEd(exp)
		}
		ctx.quiet = oquiet
		ctx.Args = ctx.Args[:0]
		ctx.Macro = curMacro
		ctx.Args = curArgs
	}
}

// ParagraphBreak represents different kinds of paragraph breaks.
type ParagraphBreak int

// Those constants represent different kinds of paragraphs breaks: normal ones,
// soft ones next to blocks boundaries, as well as paragraph breaks in lists
// and tables before new items.
const (
	ParBreakNormal ParagraphBreak = iota
	ParBreakBlock
	ParBreakItem
	ParBreakForced
)

// endEventualParagraph ends an eventual paragraph. The break can be “soft”
// (for example in LaTeX this allows a list to belong to a surrounding
// paragraph).
func endParagraph(exp Exporter, pbreak ParagraphBreak) {
	ctx := exp.Context()
	if ctx.parScope {
		processParagraph(exp)
		exp.EndParagraph(pbreak)
	}
}

func processParagraph(exp Exporter) {
	ctx := exp.Context()
	_, err := ctx.Wout.Write(exp.FormatParagraph(ctx.buf.Bytes()))
	if err != nil {
		ctx.Errorf("writing paragraph output: %v", err)
	}
	ctx.buf.Reset()
	ctx.parScope = false
}

// warnUnclosedScope warns for an unclosed scope of some kind.
func warnUnclosedScope(exp Exporter, scope *scope) {
	ctx := exp.Context()
	beginmacro := scope.macro
	var endmacro string
	switch beginmacro {
	case "Bm":
		endmacro = "Em"
	case "Bl", "It":
		endmacro = "El"
	case "Bd":
		endmacro = "Ed"
	case "#if":
		endmacro = "#;"
	}
	location := ctx.scopeLocation(scope)
	var tag string
	if scope.tag != "" {
		tag = " of type " + scope.tag
	}
	var msg string
	var m = ctx.Macro
	if !ctx.Inline {
		msg = fmt.Sprintf("found %s while `.%s' macro%s %s isn't closed yet by a `.%s'",
			m, beginmacro, tag, location, endmacro)
	} else {
		var inUserMacroMsg string
		if scope.inUserMacro {
			inUserMacroMsg = " in user macro"
		}
		msg = fmt.Sprintf("unclosed inline markup block%s%s", tag, inUserMacroMsg)
	}
	ctx.Error(msg)
}

// checkForUnclosedFormatBlock searches for an unclosed Bf block, and warns
// about it.
func checkForUnclosedFormatBlock(exp Exporter) {
	ctx := exp.Context()
	if ctx.bfInfo == nil {
		return
	}
	var file string
	if ctx.loc.curFile != ctx.bfInfo.file {
		file = " of file " + ctx.bfInfo.file
	}
	var inUserMacro string
	if ctx.bfInfo.inUserMacro {
		inUserMacro = " opened inside user macro"
	}
	msg := fmt.Sprintf("found `%s' while `.Bf' macro%s at line %d%s isn't closed by a `.Ef'",
		ctx.Macro, inUserMacro, ctx.bfInfo.line, file)
	ctx.Error(msg)
}

// checkForUnclosedDe checks for an unterminated user macro definition and warns about it.
func checkForUnclosedDe(exp Exporter) {
	ctx := exp.Context()
	if ctx.uMacroDef == nil {
		return
	}
	ctx.Errorf("found End Of File while `.#de' macro at line %d of file %s isn't closed by a `.#.'", ctx.uMacroDef.line, ctx.uMacroDef.file)
}

func scopeVerse(exp Exporter) bool {
	ctx := exp.Context()
	scopes, ok := ctx.scopes[scopeBlock]
	if ok && len(scopes) > 0 {
		s := scopes[len(scopes)-1]
		return s.macro == "Bl" && s.tag == "verse"
	}
	return false
}

func scopeIt(exp Exporter) bool {
	ctx := exp.Context()
	scopes := ctx.scopes[scopeBlock]
	if len(scopes) > 0 {
		s := scopes[len(scopes)-1]
		return s.macro == "It"
	}
	return false
}
