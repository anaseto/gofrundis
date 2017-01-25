package mom

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/escape"
	"github.com/anaseto/gofrundis/frundis"
)

type Options struct {
	OutputFile string // name of output file or directory
	Standalone bool   // generate complete document with headers
}

// NewExporter returns a frundis.Exporter suitable to produce groff mom.
// See type Options for options.
func NewExporter(opts *Options) frundis.Exporter {
	return &exporter{
		OutputFile: opts.OutputFile,
		Standalone: opts.Standalone}
}

type exporter struct {
	Bctx          *frundis.BaseContext
	Ctx           *frundis.Context
	OutputFile    string
	curOutputFile *os.File
	Standalone    bool
	dominilof     bool
	dominilot     bool
	dominitoc     bool
	minitoc       bool
	verse         bool
	inCell        bool
	fontstack     []string
}

func (exp *exporter) Init() {
	bctx := &frundis.BaseContext{Format: "mom"}
	exp.Bctx = bctx
	bctx.Init()
	ctx := &frundis.Context{W: bufio.NewWriter(os.Stdout)}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.Roff
}

func (exp *exporter) Reset() error {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	ctx.Reset()
	bctx.Reset()
	if exp.OutputFile != "" {
		var err error
		exp.curOutputFile, err = os.Create(exp.OutputFile)
		if err != nil {
			return errors.New(fmt.Sprintf("frundis:%v\n", err))
		}
	}
	if exp.curOutputFile == nil {
		exp.curOutputFile = os.Stdout
	}
	ctx.W = bufio.NewWriter(exp.curOutputFile)
	if exp.Standalone {
		exp.beginMomDocument()
	}
	return nil
}

func (exp *exporter) PostProcessing() {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	ctx.W.Flush()
	if exp.curOutputFile != nil {
		err := exp.curOutputFile.Close()
		if err != nil {
			bctx.Error(err)
		}
	}
}

func (exp *exporter) BaseContext() *frundis.BaseContext {
	return exp.Bctx
}

func (exp *exporter) BlockHandler() {
	frundis.BlockHandler(exp)
}

func (exp *exporter) BeginDescList() {
	exp.Context().W.WriteString(".LIST USER \"\"\n")
}

func (exp *exporter) BeginDescValue() {
	// do nothing
}

func (exp *exporter) BeginDialogue() {
	ctx := exp.Context()
	w := ctx.GetW()
	dmark, ok := ctx.Params["dmark"]
	if !ok {
		dmark = "–"
	} else {
		dmark = escape.Roff(dmark)
	}
	fmt.Fprint(w, dmark)
}

func (exp *exporter) BeginDisplayBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
	if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
	}
	if tag != "" {
		dtag, ok := ctx.Dtags[tag]
		var cmd string
		if ok {
			cmd = dtag.Cmd
		}
		if cmd != "" {
			fmt.Fprintf(w, ".%s\n", cmd)
		}
	}
}

func (exp *exporter) BeginEnumList() {
	exp.Context().W.WriteString(".LIST\n")
}

func (exp *exporter) BeginHeader(macro string, title string, numbered bool, renderedTitle string) {
	ctx := exp.Context()
	level := 1
	switch macro {
	case "Ch":
		level = 2
	case "Sh":
		level = 3
	case "Ss":
		level = 4
	}
	toc, _ := ctx.LoXInfo["toc"]
	entry, _ := toc[title]
	fmt.Fprintf(ctx.GetW(), ".HEADING %d NAMED s:%d \"", level, entry.Count)
}

func (exp *exporter) BeginItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, ".ITEM\n")
}

func (exp *exporter) BeginEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, ".ITEM\n")
}

func (exp *exporter) BeginItemList() {
	exp.Context().W.WriteString(".LIST\n")
}

func (exp *exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
	if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
	}
	mtag, okMtag := ctx.Mtags[tag]
	exp.fontstack = append(exp.fontstack, mtag.Cmd)
	if !okMtag {
		fmt.Fprint(w, "\\f[I]")
	} else {
		if mtag.Cmd != "" {
			fmt.Fprintf(w, "\\f[%s]", mtag.Cmd)
		}
	}
	if okMtag {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *exporter) BeginParagraph() {
	// nothing to do
}

func (exp *exporter) BeginPhrasingMacroInParagraph(nospace bool) {
	frundis.BeginPhrasingMacroInParagraph(exp, nospace)
}

func (exp *exporter) BeginTable(title string, count int, ncols int) {
	w := exp.Context().GetW()
	lll := strings.Repeat("l ", ncols)
	if title != "" {
		fmt.Fprintf(w, ".FLOAT\n")
	}
	fmt.Fprintf(w, ".TS\nallbox;\n%s.\n", lll)
}

func (exp *exporter) BeginTableCell() {
	ctx := exp.Context()
	if ctx.TableCell > 1 {
		fmt.Fprint(ctx.GetW(), "\t")
	}
	exp.inCell = true
}

func (exp *exporter) BeginTableRow() {
}

func (exp *exporter) BeginVerse(title string, count int) {
	w := exp.Context().GetW()
	exp.verse = true
	if title != "" {
		fmt.Fprintf(w, ".HEADING 5 \"%s\"\n", title)
		// fmt.Fprintf(w, "\\label{poem:%d}\n", count) // TODO
	}
	fmt.Fprint(w, ".QUOTE\n")
}

func (exp *exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing to be done for now
}

func (exp *exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *exporter) CrossReference(id string, name string, loXentry *frundis.LoXinfo, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	switch {
	case loXentry != nil:
		fmt.Fprintf(w, ".PDF_LINK \"%s:%d\" SUFFIX \"%s\" \"%s\"", loXentry.Ref, loXentry.Count, punct, name)
		// FIXME?: name could mess with surrounding markup if it has
		// markup (the problem is groff \f[..] that cannot simply be
		// reliably closed).
	case id != "":
		ref, _ := ctx.IDs[id] // we know that it's ok
		fmt.Fprintf(w, ".PDF_LINK \"%s\" SUFFIX \"%s\" \"%s\"", ref, punct, name)
	default:
		fmt.Fprintf(w, "%s%s", name, punct)
	}
}

func (exp *exporter) DescName(name string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, ".ITEM\n\\f[B]%s\\f[R]\n", name)
}

func (exp *exporter) EndDescList() {
	exp.Context().W.WriteString(".LIST OFF\n")
}

func (exp *exporter) EndDescValue() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndDisplayBlock(tag string) {
	if tag != "" {
		ctx := exp.Context()
		w := ctx.GetW()
		dtag, ok := ctx.Dtags[tag]
		var cmd string
		if ok {
			cmd = dtag.Cmd
		}
		if cmd != "" {
			fmt.Fprintf(w, ".%s OFF\n", cmd)
		}
	}
}

func (exp *exporter) EndEnumList() {
	exp.Context().W.WriteString(".LIST OFF\n")
}

func (exp *exporter) EndEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndHeader(macro string, title string, numbered bool, titleText string) {
	// TODO: rethink args (pass loxinfo?)
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "\"\n")
	// if !numbered {
	// 	fmt.Fprintf(w, "\\addcontentsline{toc}{%s}{%s}\n", cmd, titleText)
	// }
	//toc, _ := ctx.LoXInfo["toc"]
	//entry, _ := toc[title]
	//fmt.Fprintf(w, ".PDF_TARGET \"s:%d\"\n", entry.Count)
}

func (exp *exporter) EndItemList() {
	exp.Context().W.WriteString(".LIST OFF\n")
}

func (exp *exporter) EndItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	mtag, okMtag := ctx.Mtags[tag]
	if okMtag {
		fmt.Fprint(w, mtag.End)
	}
	exp.fontstack = exp.fontstack[:len(exp.fontstack)-1]
	cmd := "R"
	if len(exp.fontstack) > 0 {
		cmd = exp.fontstack[len(exp.fontstack)-1]
	}
	fmt.Fprintf(w, "\\f[%s]%s", cmd, punct)
}

func (exp *exporter) EndParagraph() {
	w := exp.Context().GetW()
	if exp.verse {
		fmt.Fprint(w, "\n\n")
	} else {
		fmt.Fprint(w, "\n.PP\n")
	}
}

func (exp *exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *exporter) EndParagraphUnsoftly() {
	w := exp.Context().GetW()
	if exp.verse {
		fmt.Fprint(w, "\n\n")
	} else {
		fmt.Fprint(w, ".PP\n")
	}
}

func (exp *exporter) EndTable(tableinfo *frundis.TableInfo) {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, ".TE\n")
	if tableinfo != nil {
		fmt.Fprintf(w, ".CAPTION \"%s\" TO_LIST TABLES\n", tableinfo.Title)
		fmt.Fprintf(w, ".FLOAT OFF\n")
		//	fmt.Fprintf(w, "\\label{tbl:%d}\n", ctx.TableCount)
	}
}

func (exp *exporter) EndTableCell() {
	exp.inCell = false
}

func (exp *exporter) EndTableRow() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndVerse() {
	w := exp.Context().GetW()
	exp.verse = false
	fmt.Fprint(w, ".QUOTE OFF\n")
}

func (exp *exporter) EndVerseLine() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) FormatParagraph(text []byte) []byte {
	if exp.inCell {
		return bytes.Replace(text, []byte("\n"), []byte(" "), -1)
	}
	return text // TODO: do something?
}

func (exp *exporter) FigureImage(image string, label string, link string) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	w := ctx.GetW()
	_, err := os.Stat(image)
	if err != nil {
		bctx.Error("image not found:", image)
		return
	}
	ext := path.Ext(image)
	if ext != ".eps" && ext != ".pdf" {
		bctx.Error("expected .eps or .pdf but got ", image)
	}
	image = escape.Roff(image)
	fmt.Fprintf(w, ".FLOAT\n")
	fmt.Fprintf(w, ".PDF_IMAGE \"%s\"\n", image)
	fmt.Fprintf(w, ".CAPTION \"%s\" TO_LIST FIGURES\n", label)
	fmt.Fprintf(w, ".FLOAT OFF\n")
	// fmt.Fprintf(w, "\\label{fig:%d}\n", ctx.FigCount)
}

func (exp *exporter) GenRef(prefix string, id string, hasfile bool) string {
	// XXX not used yet
	if prefix == "" {
		return fmt.Sprintf("%s", id)
	} else {
		return fmt.Sprintf("%s", prefix)
	}
}

func (exp *exporter) HeaderReference(macro string) string {
	// XXX not used yet
	return "s"
}

func (exp *exporter) InlineImage(image string, link string, punct string) {
	bctx := exp.BaseContext()
	if strings.ContainsAny(image, "{}") {
		bctx.Error("path argument and label should not contain the characters `{', or `}")
		return
	}
	w := exp.Context().GetW()
	_, err := os.Stat(image)
	if err != nil {
		bctx.Error("image not found:", image)
		return
	}
	ext := path.Ext(image)
	if ext != ".eps" && ext != ".pdf" {
		bctx.Error("expected .eps or .pdf but got ", image)
	}
	image = escape.Roff(image)
	fmt.Fprintf(w, ".PDF_IMAGE \"%s\"", image) // TODO: use punct
}

func (exp *exporter) LkWithLabel(uri string, label string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.Roff(parsedURL.String())
	}
	fmt.Fprintf(w, ".PDF_WWW_LINK %s SUFFIX \"%s\" \"%s\"", u, punct, escape.Roff(label))
	// XXX warn if label ends with '*' or '+' ? (they have special meaning
	// in mom at the end of the hotlink text)
}

func (exp *exporter) LkWithoutLabel(uri string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.Roff(parsedURL.String())
	}
	fmt.Fprintf(w, ".PDF_WWW_LINK %s SUFFIX \"%s\"", u, punct)
	// XXX: -ns option should be invalid after...
}

func (exp *exporter) ParagraphTitle(title string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, ".HEADING 5 PARAHEAD \"%s\"\n", title)
}

func (exp *exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return escape.Roff(exp.BaseContext().InlinesToText(text))
}

func (exp *exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	w := exp.Context().GetW()
	bctx := exp.BaseContext()
	switch {
	case flags["lof"]:
		fmt.Fprint(w, ".LIST_OF_FIGURES\n")
	case flags["lot"]:
		fmt.Fprint(w, ".LIST_OF_TABLES\n")
	case flags["lop"]:
		// XXX: do something about this?
		bctx.Error("list of poems not available for mom")
	default:
		fmt.Fprint(w, ".TOC\n")
	}
}

func (exp *exporter) TableOfContentsInfos(flags map[string]bool) {
}

func (exp *exporter) Xdtag(cmd string) frundis.Dtag {
	return frundis.Dtag{Cmd: cmd}
}

func (exp *exporter) Xftag(shell string) frundis.Ftag {
	return frundis.Ftag{Shell: shell}
}

func (exp *exporter) Xmtag(cmd *string, begin string, end string) frundis.Mtag {
	var c string
	if cmd == nil {
		c = "I"
	} else {
		c = *cmd
	}
	return frundis.Mtag{Begin: escape.Roff(begin), End: escape.Roff(end), Cmd: c}
}
