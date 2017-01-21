package tpl

import (
	"bufio"
	"fmt"
	"html"
	"os"

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
	escape        func(string) string
}

func (exp *Exporter) Init() {
	bctx := &frundis.BaseContext{Format: exp.Format}
	exp.Bctx = bctx
	bctx.Init()
	ctx := &frundis.Context{W: bufio.NewWriter(os.Stdout)}
	exp.Ctx = ctx
	ctx.Init()
	switch bctx.Format {
	case "xhtml":
		ctx.Filters["escape"] = html.EscapeString
	case "latex":
		ctx.Filters["escape"] = escape.LaTeX
	}
	f, ok := ctx.Filters["escape"]
	if ok {
		exp.escape = f
	} else {
		exp.escape = func(s string) string { return s }
	}
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
	frundis.MinimalBlockHandler(exp)
}

func (exp *Exporter) BeginDescList() {
}

func (exp *Exporter) BeginDescValue() {
}

func (exp *Exporter) BeginDialogue() {
}

func (exp *Exporter) BeginDisplayBlock(tag string, id string) {
}

func (exp *Exporter) BeginEnumList() {
}

func (exp *Exporter) BeginHeader(macro string, title string, numbered bool, renderedText string) {
}

func (exp *Exporter) BeginItem() {
}

func (exp *Exporter) BeginEnumItem() {
}

func (exp *Exporter) BeginItemList() {
}

func (exp *Exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
	mtag, ok := ctx.Mtags[tag]
	if ok {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *Exporter) BeginParagraph() {
}

func (exp *Exporter) BeginPhrasingMacroInParagraph(nospace bool) {
	frundis.BeginPhrasingMacroInParagraph(exp, nospace)
}

func (exp *Exporter) BeginTable(title string, count int, ncols int) {
}

func (exp *Exporter) BeginTableCell() {
}

func (exp *Exporter) BeginTableRow() {
}

func (exp *Exporter) BeginVerse(title string, count int) {
}

func (exp *Exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing for now
}

func (exp *Exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *Exporter) CrossReference(id string, name string, loXentry *frundis.LoXinfo, punct string) {
}

func (exp *Exporter) DescName(name string) {
}

func (exp *Exporter) EndDescList() {
}

func (exp *Exporter) EndDescValue() {
}

func (exp *Exporter) EndDisplayBlock(tag string) {
}

func (exp *Exporter) EndEnumList() {
}

func (exp *Exporter) EndEnumItem() {
}

func (exp *Exporter) EndHeader(macro string, title string, numbered bool, titleText string) {
}

func (exp *Exporter) EndItemList() {
}

func (exp *Exporter) EndItem() {
}

func (exp *Exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	mtag, ok := ctx.Mtags[tag]
	if ok {
		fmt.Fprint(w, mtag.End)
	}
	fmt.Fprint(w, punct)
}

func (exp *Exporter) EndParagraph() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *Exporter) EndParagraphUnsoftly() {
	// do nothing
}

func (exp *Exporter) EndTable(tableinfo *frundis.TableInfo) {
}

func (exp *Exporter) EndTableCell() {
}

func (exp *Exporter) EndTableRow() {
}

func (exp *Exporter) EndVerse() {
}

func (exp *Exporter) EndVerseLine() {
}

func (exp *Exporter) FormatParagraph(text []byte) []byte {
	return text
}

func (exp *Exporter) FigureImage(image string, label string, link string) {
}

func (exp *Exporter) GenRef(prefix string, id string, hasfile bool) string {
	return ""
}

func (exp *Exporter) HeaderReference(macro string) string {
	return ""
}

func (exp *Exporter) InlineImage(image string, link string, punct string) {
}

func (exp *Exporter) LkWithLabel(url string, label string, punct string) {
}

func (exp *Exporter) LkWithoutLabel(url string, punct string) {
}

func (exp *Exporter) ParagraphTitle(title string) {
}

func (exp *Exporter) RenderText(text []ast.Inline) string {
	return exp.escape(exp.BaseContext().InlinesToText(text))
}

func (exp *Exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
}

func (exp *Exporter) TableOfContentsInfos(flags map[string]bool) {
}

func (exp *Exporter) Xdtag(cmd string) frundis.Dtag {
	return frundis.Dtag{}
}

func (exp *Exporter) Xftag(shell string) frundis.Ftag {
	return frundis.Ftag{Shell: shell}
}

func (exp *Exporter) Xmtag(cmd *string, begin string, end string) frundis.Mtag {
	return frundis.Mtag{Begin: begin, End: end}
}
