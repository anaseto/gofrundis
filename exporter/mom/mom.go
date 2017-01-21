package mom

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/escape"
	"github.com/anaseto/gofrundis/frundis"
)

type Exporter struct {
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
	fontstack     []string
}

func (exp *Exporter) Init() {
	bctx := &frundis.BaseContext{Format: "mom"}
	exp.Bctx = bctx
	bctx.Init()
	ctx := &frundis.Context{W: bufio.NewWriter(os.Stdout)}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.Roff
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
	if exp.Standalone {
		exp.beginMomDocument()
	}
}

func (exp *Exporter) PostProcessing() {
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

func (exp *Exporter) BaseContext() *frundis.BaseContext {
	return exp.Bctx
}

func (exp *Exporter) BlockHandler() {
	frundis.BlockHandler(exp)
}

func (exp *Exporter) BeginDescList() {
	exp.Context().W.WriteString(".LIST USER \"\"\n")
}

func (exp *Exporter) BeginDescValue() {
	// do nothing
}

func (exp *Exporter) BeginDialogue() {
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

func (exp *Exporter) BeginDisplayBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
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
	//if id != "" { // TODO: not sure how to do this
	//	fmt.Fprintf(w, "\\hypertarget{%s}{}\n", id)
	//}
}

func (exp *Exporter) BeginEnumList() {
	exp.Context().W.WriteString(".LIST\n")
}

func (exp *Exporter) BeginHeader(macro string, title string, numbered bool, renderedTitle string) {
	ctx := exp.Context()
	cmd := ctx.TocInfo.HeaderLevel(macro)
	fmt.Fprintf(ctx.GetW(), ".HEADING %d \"", cmd)
}

func (exp *Exporter) BeginItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, ".ITEM\n")
}

func (exp *Exporter) BeginEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, ".ITEM\n")
}

func (exp *Exporter) BeginItemList() {
	exp.Context().W.WriteString(".LIST\n")
}

func (exp *Exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
	//if id != "" { // XXX not sure how this works with groff
	//	fmt.Fprintf(w, "\\hypertarget{%s}{", id)
	//}
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

func (exp *Exporter) BeginParagraph() {
	// nothing to do
}

func (exp *Exporter) BeginPhrasingMacroInParagraph(nospace bool) {
	frundis.BeginPhrasingMacroInParagraph(exp, nospace)
}

func (exp *Exporter) BeginTable(title string, count int, ncols int) {
	w := exp.Context().GetW()
	lll := strings.Repeat("l ", ncols)
	if title != "" {
		fmt.Fprintf(w, ".FLOAT\n")
	}
	fmt.Fprintf(w, ".TS\nallbox;\n%s.\n", lll)
}

func (exp *Exporter) BeginTableCell() {
	ctx := exp.Context()
	if ctx.TableCell > 1 {
		fmt.Fprint(ctx.GetW(), "\t")
	}
}

func (exp *Exporter) BeginTableRow() {
}

func (exp *Exporter) BeginVerse(title string, count int) {
	w := exp.Context().GetW()
	exp.verse = true
	if title != "" {
		fmt.Fprintf(w, ".HEADING 5 \"%s\"\n", title)
		// fmt.Fprintf(w, "\\label{poem:%d}\n", count) // TODO
	}
	fmt.Fprint(w, ".QUOTE\n")
}

func (exp *Exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing to be done for now
}

func (exp *Exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *Exporter) CrossReference(id string, name string, loXentry *frundis.LoXinfo, punct string) {
	// TODO: not sure how to do this with groff
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprintf(w, "%s%s", name, punct)
}

func (exp *Exporter) DescName(name string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, ".ITEM\n\\f[B]%s\\f[R]\n", name)
}

func (exp *Exporter) EndDescList() {
	exp.Context().W.WriteString(".LIST OFF\n")
}

func (exp *Exporter) EndDescValue() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndDisplayBlock(tag string) {
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

func (exp *Exporter) EndEnumList() {
	exp.Context().W.WriteString(".LIST OFF\n")
}

func (exp *Exporter) EndEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndHeader(macro string, title string, numbered bool, titleText string) {
	// TODO: rethink args (pass loxinfo?)
	ctx := exp.Context()
	w := ctx.GetW()
	// cmd := momHeaderName(macro)
	fmt.Fprint(w, "\"\n")
	// if !numbered {
	// 	fmt.Fprintf(w, "\\addcontentsline{toc}{%s}{%s}\n", cmd, titleText)
	// }
	// toc, _ := ctx.LoXInfo["toc"]
	// entry, _ := toc[title]
	// fmt.Fprintf(w, "\\label{s:%d}\n", entry.Count)
}

func (exp *Exporter) EndItemList() {
	exp.Context().W.WriteString(".LIST OFF\n")
}

func (exp *Exporter) EndItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndMarkupBlock(tag string, id string, punct string) {
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

func (exp *Exporter) EndParagraph() {
	w := exp.Context().GetW()
	if exp.verse {
		fmt.Fprint(w, "\n\n")
	} else {
		fmt.Fprint(w, "\n.PP\n")
	}
}

func (exp *Exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *Exporter) EndParagraphUnsoftly() {
	w := exp.Context().GetW()
	if exp.verse {
		fmt.Fprint(w, "\n\n")
	} else {
		fmt.Fprint(w, ".PP\n")
	}
}

func (exp *Exporter) EndTable(tableinfo *frundis.TableInfo) {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, ".TE\n")
	if tableinfo != nil {
		fmt.Fprintf(w, ".CAPTION \"%s\" TO_LIST TABLES\n", tableinfo.Title)
		fmt.Fprintf(w, ".FLOAT OFF\n")
		//	fmt.Fprintf(w, "\\label{tbl:%d}\n", ctx.TableCount)
	}
}

func (exp *Exporter) EndTableCell() {
}

func (exp *Exporter) EndTableRow() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndVerse() {
	w := exp.Context().GetW()
	exp.verse = false
	fmt.Fprint(w, ".QUOTE OFF\n")
}

func (exp *Exporter) EndVerseLine() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) FormatParagraph(text []byte) []byte {
	return text // TODO: do something?
}

func (exp *Exporter) FigureImage(image string, label string, link string) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	w := ctx.GetW()
	_, err := os.Stat(image)
	if err != nil {
		bctx.Error("image not found:", image)
		return
	}
	ext := path.Ext(image)
	if ext != "eps" && ext != "pdf" {
		bctx.Error("expected .eps or .pdf but got ", image)
	}
	image = escape.Roff(image)
	fmt.Fprintf(w, ".FLOAT\n")
	fmt.Fprintf(w, ".PDF_IMAGE \"%s\"\n", image)
	fmt.Fprintf(w, ".CAPTION \"%s\" TO_LIST FIGURES\n", label)
	fmt.Fprintf(w, ".FLOAT OFF\n")
	// fmt.Fprintf(w, "\\label{fig:%d}\n", ctx.FigCount)
}

func (exp *Exporter) GenRef(prefix string, id string, hasfile bool) string {
	// XXX not used yet
	if prefix == "" {
		return fmt.Sprintf("%s", id)
	} else {
		return fmt.Sprintf("%s", prefix)
	}
}

func (exp *Exporter) HeaderReference(macro string) string {
	// XXX not used yet
	return "s"
}

func (exp *Exporter) InlineImage(image string, link string, punct string) {
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
	image = escape.Roff(image)
	fmt.Fprintf(w, ".PDF_IMAGE \"%s\"\n", image) // TODO: use punct
}

func (exp *Exporter) LkWithLabel(uri string, label string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.Roff(parsedURL.String())
	}
	fmt.Fprintf(w, "%s (%s)%s", escape.Roff(label), u, punct)
}

func (exp *Exporter) LkWithoutLabel(uri string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.Roff(parsedURL.String())
	}
	fmt.Fprintf(w, "%s%s", u, punct) // XXX italic or something?
}

func (exp *Exporter) ParagraphTitle(title string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, ".HEADING 5 PARAHEAD \"%s\"\n", title)
}

func (exp *Exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return escape.Roff(exp.BaseContext().InlinesToText(text))
}

func (exp *Exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
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

func (exp *Exporter) TableOfContentsInfos(flags map[string]bool) {
}

func (exp *Exporter) Xdtag(cmd string) frundis.Dtag {
	return frundis.Dtag{Cmd: cmd}
}

func (exp *Exporter) Xftag(shell string) frundis.Ftag {
	return frundis.Ftag{Shell: shell}
}

func (exp *Exporter) Xmtag(cmd *string, begin string, end string) frundis.Mtag {
	var c string
	if cmd == nil {
		c = "I"
	} else {
		c = *cmd
	}
	return frundis.Mtag{Begin: escape.Roff(begin), End: escape.Roff(end), Cmd: c}
}
