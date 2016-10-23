package markdown

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/escape"
	"github.com/anaseto/gofrundis/frundis"
)

type Exporter struct {
	Bctx          *frundis.BaseContext
	Ctx           *frundis.Context
	Format        string
	OutputFile    string
	curOutputFile *os.File
	nesting       int
	verse         bool
}

func (exp *Exporter) Init() {
	bctx := &frundis.BaseContext{Format: "markdown"}
	exp.Bctx = bctx
	bctx.Init()
	ctx := &frundis.Context{W: bufio.NewWriter(os.Stdout)}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.EscapeMarkdownString
}

func (exp *Exporter) Reset() {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	ctx.Reset()
	bctx.Reset()
	if exp.OutputFile != "" {
		var err error
		exp.curOutputFile, err = os.Create(exp.OutputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "frundis:%v\n", err)
			os.Exit(1)
		}
	}
	if exp.curOutputFile == nil {
		exp.curOutputFile = os.Stdout
	}
	ctx.W = bufio.NewWriter(exp.curOutputFile)
}

func (exp *Exporter) PostProcessing() {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	ctx.W.Flush()
	if exp.curOutputFile != nil {
		err := exp.curOutputFile.Close()
		if err != nil {
			bctx.Error(err)
		}
	}
}

func (exp *Exporter) BaseContext() *frundis.BaseContext {
	return exp.Bctx
}

func (exp *Exporter) BlockHandler() {
	frundis.BlockHandler(exp)
}

func (exp *Exporter) BeginDescList() {
}

func (exp *Exporter) BeginDescValue() {
	exp.nesting = 3
	exp.Context().W.WriteString("  ~ ")
}

func (exp *Exporter) BeginDialogue() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "—")
}

func (exp *Exporter) BeginDisplayBlock(tag string, id string) {
}

func (exp *Exporter) BeginEnumList() {
	w := exp.Context().GetW()
	if exp.nesting > 0 {
		fmt.Fprint(w, "\n")
	}
	exp.nesting += 3
}

func (exp *Exporter) BeginHeader(macro string, title string, numbered bool, renderedText string) {
	ctx := exp.Context()
	w := ctx.GetW()
	num := ctx.TocInfo.HeaderLevel(macro)
	switch num {
	case 3:
		fmt.Fprint(w, "### ")
	case 4:
		fmt.Fprint(w, "#### ")
	}
}

func (exp *Exporter) BeginItem() {
	w := exp.Context().GetW()
	var item string
	if exp.nesting%6 == 0 {
		item = "* "
	} else if exp.nesting%4 == 0 {
		item = "+ "
	} else {
		item = "- "
	}
	if exp.nesting >= 2 {
		// should allways be the case
		fmt.Fprint(w, strings.Repeat(" ", exp.nesting-2))
	} else {
		bctx := exp.BaseContext()
		bctx.Error("unexpected nesting")
	}
	fmt.Fprint(w, item)
}

func (exp *Exporter) BeginEnumItem() {
	w := exp.Context().GetW()
	var item string
	item = "1. "
	if exp.nesting >= 3 {
		// should allways be the case
		fmt.Fprint(w, strings.Repeat(" ", exp.nesting-3))
	} else {
		bctx := exp.BaseContext()
		bctx.Error("unexpected nesting")
	}
	fmt.Fprint(w, item)
}

func (exp *Exporter) BeginItemList() {
	w := exp.Context().GetW()
	if exp.nesting > 0 {
		fmt.Fprint(w, "\n")
	}
	exp.nesting += 2
}

func (exp *Exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
	mtag, ok := ctx.Mtags[tag]
	if !ok {
		fmt.Fprint(w, "*")
	} else {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *Exporter) BeginParagraph() {
}

func (exp *Exporter) BeginTable(title string, count int, ncols int) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n") // XXX bof
}

func (exp *Exporter) BeginTableCell() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\t")
}

func (exp *Exporter) BeginTableRow() {
}

func (exp *Exporter) BeginVerse(title string, count int) {
	w := exp.Context().GetW()
	exp.verse = true
	fmt.Fprint(w, "##### "+title+"\n\n")
}

func (exp *Exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing for now
}

func (exp *Exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *Exporter) CrossReference(id string, name string, loXentry *frundis.LoXinfo, punct string) {
	w := exp.Context().GetW()
	// TODO: do some kind of cross-references (pandoc-like ?)
	fmt.Fprintf(w, "%s%s", name, punct)
}

func (exp *Exporter) DescName(name string) {
	w := exp.Context().GetW()
	fmt.Fprint(w, name)
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndDescList() {
	exp.Context().W.WriteString("\n")
}

func (exp *Exporter) EndDescValue() {
	w := exp.Context().GetW()
	exp.nesting = 0
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndDisplayBlock(tag string) {
}

func (exp *Exporter) EndEnumList() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
	exp.nesting -= 3
	if exp.nesting == 0 {
		fmt.Fprint(w, "<!-- -->\n\n")
	}
}

func (exp *Exporter) EndEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndHeader(macro string, title string, numbered bool, titleText string) {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "\n")
	var uc string
	num := ctx.TocInfo.HeaderLevel(macro)
	switch num {
	case 1:
		uc = "="
	case 2:
		uc = "-"
	}
	if num <= 2 {
		fmt.Fprint(w, strings.Repeat(uc, utf8.RuneCountInString(titleText)))
		fmt.Fprint(w, "\n")
	}
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndItemList() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
	exp.nesting -= 2
	if exp.nesting == 0 {
		fmt.Fprint(w, "<!-- -->\n\n")
	}
}

func (exp *Exporter) EndItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	mtag, ok := ctx.Mtags[tag]
	if !ok {
		fmt.Fprint(w, "*")
	} else {
		fmt.Fprint(w, mtag.End)
	}
	fmt.Fprint(w, punct)
}

func (exp *Exporter) EndParagraph() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n\n")
}

func (exp *Exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *Exporter) EndParagraphUnsoftly() {
	// do nothing
}

func (exp *Exporter) EndTable(tableinfo *frundis.TableInfo) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndTableCell() {
	// do nothing
}

func (exp *Exporter) EndTableRow() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndVerse() {
	w := exp.Context().GetW()
	exp.verse = false
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndVerseLine() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\\\n")
}

func (exp *Exporter) FormatParagraph(text []byte) []byte {
	var indent int
	if exp.nesting > 0 {
		indent = exp.nesting
	}
	if exp.verse {
		return text
	}
	return processText(indent, text)
}

func (exp *Exporter) FigureImage(image string, label string, link string) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "!["+label+"]"+"("+image+")")
}

func (exp *Exporter) GenRef(prefix string, id string, hasfile bool) string {
	// XXX useless for now
	return ""
}

func (exp *Exporter) HeaderReference(macro string) string {
	// XXX useless for now
	return ""
}

func (exp *Exporter) InlineImage(image string, link string, punct string) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "!["+image+"]"+"("+image+")"+punct)
}

func (exp *Exporter) LkWithLabel(url string, label string, punct string) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "["+label+"]"+"("+url+")"+punct)
}

func (exp *Exporter) LkWithoutLabel(url string, punct string) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<"+url+">"+punct)
}

func (exp *Exporter) ParagraphTitle(title string) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "**"+title+"**. ")
}

func (exp *Exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return escape.EscapeMarkdownString(exp.BaseContext().InlinesToText(text))
}

func (exp *Exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	// TODO ? (TOC probably not very useful here)
}

func (exp *Exporter) TableOfContentsInfos(flags map[string]bool) {
}

func (exp *Exporter) Xdtag(cmd string) frundis.Dtag {
	return frundis.Dtag{Cmd: cmd}
}

func (exp *Exporter) Xftag(shell string) frundis.Ftag {
	return frundis.Ftag{Shell: shell}
}

func (exp *Exporter) Xmtag(cmd *string, begin string, end string) frundis.Mtag {
	return frundis.Mtag{Begin: begin, End: end}
}
