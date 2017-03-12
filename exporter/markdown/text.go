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

// Options gathers configuration for markdown exporter.
type Options struct {
	OutputFile string // name of output file or directory
}

// NewExporter returns a frundis.Exporter suitable to produce markdown.
// See type Options for options.
func NewExporter(opts *Options) frundis.Exporter {
	return &exporter{OutputFile: opts.OutputFile}
}

type exporter struct {
	Bctx          *frundis.Context
	Ctx           *frundis.Context
	Format        string
	OutputFile    string
	curOutputFile *os.File
	nesting       int
	verse         bool
}

func (exp *exporter) Init() {
	ctx := &frundis.Context{Wout: bufio.NewWriter(os.Stdout), Format: "markdown"}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.Markdown
}

func (exp *exporter) Reset() error {
	ctx := exp.Context()
	ctx.Reset()
	if exp.OutputFile != "" {
		var err error
		exp.curOutputFile, err = os.Create(exp.OutputFile)
		if err != nil {
			return fmt.Errorf("frundis:%v\n", err)
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

func (exp *exporter) BlockHandler() {
	frundis.DefaultBlockHandler(exp)
}

func (exp *exporter) BeginDescList(id string) {
}

func (exp *exporter) BeginDescValue() {
	exp.nesting = 3
	exp.Context().Wout.WriteString("  ~ ")
}

func (exp *exporter) BeginDialogue() {
	w := exp.Context().W()
	fmt.Fprint(w, "—")
}

func (exp *exporter) BeginDisplayBlock(tag string, id string) {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) BeginEnumList(id string) {
	w := exp.Context().W()
	if exp.nesting > 0 {
		fmt.Fprint(w, "\n")
	}
	exp.nesting += 3
}

func (exp *exporter) BeginHeader(macro string, numbered bool, title string) {
	ctx := exp.Context()
	w := ctx.W()
	num := ctx.Toc.HeaderLevel(macro)
	switch num {
	case 3:
		fmt.Fprint(w, "### ")
	case 4:
		fmt.Fprint(w, "#### ")
	}
}

func (exp *exporter) BeginItem() {
	ctx := exp.Context()
	w := ctx.W()
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
		ctx.Error("unexpected nesting")
	}
	fmt.Fprint(w, item)
}

func (exp *exporter) BeginEnumItem() {
	ctx := exp.Context()
	w := ctx.W()
	var item string
	item = "1. "
	if exp.nesting >= 3 {
		// should allways be the case
		fmt.Fprint(w, strings.Repeat(" ", exp.nesting-3))
	} else {
		ctx.Error("unexpected nesting")
	}
	fmt.Fprint(w, item)
}

func (exp *exporter) BeginItemList(id string) {
	w := exp.Context().W()
	if exp.nesting > 0 {
		fmt.Fprint(w, "\n")
	}
	exp.nesting += 2
}

func (exp *exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.W()
	mtag, ok := ctx.Mtags[tag]
	if !ok {
		fmt.Fprint(w, "*")
	} else {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *exporter) BeginParagraph() {
}

func (exp *exporter) BeginPhrasingMacroInParagraph(nospace bool) {
	frundis.BeginPhrasingMacroInParagraph(exp, nospace)
}

func (exp *exporter) BeginTable(tableinfo *frundis.TableData) {
	w := exp.Context().W()
	fmt.Fprint(w, "\n") // XXX bof
}

func (exp *exporter) BeginTableCell() {
	w := exp.Context().W()
	fmt.Fprint(w, "\t")
}

func (exp *exporter) BeginTableRow() {
}

func (exp *exporter) BeginVerse(title string, id string) {
	w := exp.Context().W()
	exp.verse = true
	fmt.Fprint(w, "##### "+title+"\n\n")
}

func (exp *exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing for now
}

func (exp *exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *exporter) CrossReference(idf frundis.IDInfo, name string, punct string) {
	w := exp.Context().W()
	// TODO: do some kind of cross-references (pandoc-like ?)
	fmt.Fprintf(w, "%s%s", name, punct)
}

func (exp *exporter) DescName(name string) {
	w := exp.Context().W()
	fmt.Fprint(w, name)
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndDescList() {
	exp.Context().Wout.WriteString("\n")
}

func (exp *exporter) EndDescValue() {
	w := exp.Context().W()
	exp.nesting = 0
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndDisplayBlock(tag string) {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndEnumList() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
	exp.nesting -= 3
	if exp.nesting == 0 {
		fmt.Fprint(w, "<!-- -->\n\n")
	}
}

func (exp *exporter) EndEnumItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndHeader(macro string, numbered bool, title string) {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "\n")
	var uc string
	num := ctx.Toc.HeaderLevel(macro)
	switch num {
	case 1:
		uc = "="
	case 2:
		uc = "-"
	}
	if num <= 2 {
		fmt.Fprint(w, strings.Repeat(uc, utf8.RuneCountInString(title)))
		fmt.Fprint(w, "\n")
	}
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndItemList() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
	exp.nesting -= 2
	if exp.nesting == 0 {
		fmt.Fprint(w, "<!-- -->\n\n")
	}
}

func (exp *exporter) EndItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	mtag, ok := ctx.Mtags[tag]
	if !ok {
		fmt.Fprint(w, "*")
	} else {
		fmt.Fprint(w, mtag.End)
	}
	fmt.Fprint(w, punct)
}

func (exp *exporter) EndParagraph() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n\n")
}

func (exp *exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *exporter) EndParagraphUnsoftly() {
	// do nothing
}

func (exp *exporter) EndTable(tableinfo *frundis.TableData) {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndTableCell() {
	// do nothing
}

func (exp *exporter) EndTableRow() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndVerse() {
	w := exp.Context().W()
	exp.verse = false
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndVerseLine() {
	w := exp.Context().W()
	fmt.Fprint(w, "\\\n")
}

func (exp *exporter) FormatParagraph(text []byte) []byte {
	var indent int
	if exp.nesting > 0 {
		indent = exp.nesting
	}
	if exp.verse {
		return text
	}
	return processText(indent, text)
}

func (exp *exporter) FigureImage(image string, label string, link string) {
	w := exp.Context().W()
	fmt.Fprint(w, "!["+label+"]"+"("+image+")")
}

func (exp *exporter) GenRef(prefix string, id string, hasfile bool) string {
	// XXX useless for now
	return ""
}

func (exp *exporter) HeaderReference(macro string) string {
	// XXX useless for now
	return ""
}

func (exp *exporter) InlineImage(image string, link string, id string, punct string) {
	w := exp.Context().W()
	fmt.Fprint(w, "!["+image+"]"+"("+image+")"+punct)
}

func (exp *exporter) LkWithLabel(url string, label string, punct string) {
	w := exp.Context().W()
	fmt.Fprint(w, "["+label+"]"+"("+url+")"+punct)
}

func (exp *exporter) LkWithoutLabel(url string, punct string) {
	w := exp.Context().W()
	fmt.Fprint(w, "<"+url+">"+punct)
}

func (exp *exporter) ParagraphTitle(title string) {
	w := exp.Context().W()
	fmt.Fprint(w, "**"+title+"**. ")
}

func (exp *exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return escape.Markdown(exp.Context().InlinesToText(text))
}

func (exp *exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	// TODO ? (TOC probably not very useful here)
}

func (exp *exporter) TableOfContentsInfos(flags map[string]bool) {
}

func (exp *exporter) Xdtag(cmd string, pairs []string) frundis.Dtag {
	return frundis.Dtag{Cmd: cmd}
}

func (exp *exporter) Xftag(shell string) frundis.Ftag {
	return frundis.Ftag{Shell: shell}
}

func (exp *exporter) Xmtag(cmd *string, begin string, end string, pairs []string) frundis.Mtag {
	return frundis.Mtag{Begin: begin, End: end}
}
