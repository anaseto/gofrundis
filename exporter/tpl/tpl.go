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

// Options gathers configuration for template mode exporter.
type Options struct {
	Format     string // "latex" or "xhtml"
	OutputFile string // name of output file or directory
}

// NewExporter returns a frundis.Exporter suitable to produce a LaTeX or XHTML template.
// See type Options for options.
func NewExporter(opts *Options) frundis.Exporter {
	return &exporter{
		Format:     opts.Format,
		OutputFile: opts.OutputFile}
}

type exporter struct {
	Ctx           *frundis.Context
	Format        string
	OutputFile    string
	curOutputFile *os.File
	escape        func(string) string
}

func (exp *exporter) Init() {
	ctx := &frundis.Context{Wout: bufio.NewWriter(os.Stdout), Format: exp.Format}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Macros = frundis.MinimalExporterMacros()
	switch ctx.Format {
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

func (exp *exporter) Reset() error {
	ctx := exp.Context()
	ctx.Reset()
	if exp.OutputFile != "" {
		var err error
		exp.curOutputFile, err = os.Create(exp.OutputFile)
		if err != nil {
			return fmt.Errorf("%v\n", err)
		}
	}
	if exp.curOutputFile == nil {
		exp.curOutputFile = os.Stdout
	}
	ctx.Wout = bufio.NewWriter(exp.curOutputFile)
	return nil
}

func (exp *exporter) PostProcessing() {
	ctx := exp.Context()
	ctx.Wout.Flush()
	if exp.curOutputFile != nil {
		err := exp.curOutputFile.Close()
		if err != nil {
			ctx.Error(err)
		}
	}
}

func (exp *exporter) BeginDescList(id string) {
}

func (exp *exporter) BeginDescValue() {
}

func (exp *exporter) BeginDialogue() {
}

func (exp *exporter) BeginDisplayBlock(tag string, id string) {
}

func (exp *exporter) BeginEnumList(id string) {
}

func (exp *exporter) BeginHeader(macro string, numbered bool, title string) {
}

func (exp *exporter) BeginItem() {
}

func (exp *exporter) BeginEnumItem() {
}

func (exp *exporter) BeginItemList(id string) {
}

func (exp *exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.W()
	mtag, ok := ctx.Mtags[tag]
	if ok {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *exporter) BeginParagraph() {
}

func (exp *exporter) BeginPhrasingMacroInParagraph(nospace bool) {
	frundis.BeginPhrasingMacroInParagraph(exp, nospace)
}

func (exp *exporter) BeginTable(tableinfo *frundis.TableData) {
}

func (exp *exporter) BeginTableCell() {
}

func (exp *exporter) BeginTableRow() {
}

func (exp *exporter) BeginVerse(title string, id string) {
}

func (exp *exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing for now
}

func (exp *exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *exporter) CrossReference(idf frundis.IDInfo, punct string) {
}

func (exp *exporter) DescName(name string) {
}

func (exp *exporter) EndDescList() {
}

func (exp *exporter) EndDescValue() {
}

func (exp *exporter) EndDisplayBlock(tag string) {
}

func (exp *exporter) EndEnumList() {
}

func (exp *exporter) EndEnumItem() {
}

func (exp *exporter) EndHeader(macro string, numbered bool, title string) {
}

func (exp *exporter) EndItemList() {
}

func (exp *exporter) EndItem() {
}

func (exp *exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	mtag, ok := ctx.Mtags[tag]
	if ok {
		fmt.Fprint(w, mtag.End)
	}
	fmt.Fprint(w, punct)
}

func (exp *exporter) EndParagraph() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *exporter) EndParagraphUnsoftly() {
	// do nothing
}

func (exp *exporter) EndTable(tableinfo *frundis.TableData) {
}

func (exp *exporter) EndTableCell() {
}

func (exp *exporter) EndTableRow() {
}

func (exp *exporter) EndVerse() {
}

func (exp *exporter) EndVerseLine() {
}

func (exp *exporter) FormatParagraph(text []byte) []byte {
	return text
}

func (exp *exporter) FigureImage(image string, caption string, link string) {
}

func (exp *exporter) GenRef(prefix string, id string, hasfile bool) string {
	return ""
}

func (exp *exporter) HeaderReference(macro string) string {
	return ""
}

func (exp *exporter) InlineImage(image string, link string, id string, punct string) {
}

func (exp *exporter) LkWithLabel(url string, label string, punct string) {
}

func (exp *exporter) LkWithoutLabel(url string, punct string) {
}

func (exp *exporter) ParagraphTitle(title string) {
}

func (exp *exporter) RenderText(text []ast.Inline) string {
	return exp.escape(exp.Context().InlinesToText(text))
}

func (exp *exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
}

func (exp *exporter) TableOfContentsInfos(flags map[string]bool) {
}

func (exp *exporter) Xdtag(cmd string, pairs []string) frundis.Dtag {
	return frundis.Dtag{}
}

func (exp *exporter) Xmtag(cmd *string, begin string, end string, pairs []string) frundis.Mtag {
	// NOTE: in contrast with other export formats, we don't escape begin and end.
	return frundis.Mtag{Begin: begin, End: end}
}
