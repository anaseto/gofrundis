package latex

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
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
}

func (exp *Exporter) Init() {
	bctx := &frundis.BaseContext{Format: "latex"}
	exp.Bctx = bctx
	bctx.Init()
	ctx := &frundis.Context{W: bufio.NewWriter(os.Stdout)}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.EscapeLatexString
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
		exp.beginLatexDocument()
	}
}

func (exp *Exporter) PostProcessing() {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	if exp.Standalone {
		exp.EndLatexDocument()
	}
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
	exp.Context().W.WriteString("\\begin{description}\n")
}

func (exp *Exporter) BeginDescValue() {
	// do nothing
}

func (exp *Exporter) BeginDialogue() {
	ctx := exp.Context()
	w := ctx.GetW()
	dmark, ok := ctx.Params["dmark"]
	if !ok {
		dmark = "---"
	} else {
		dmark = escape.EscapeLatexString(dmark)
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
			fmt.Fprintf(w, "\\begin{%s}\n", cmd)
		}
	}
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}\n", id)
	}
}

func (exp *Exporter) BeginEnumList() {
	exp.Context().W.WriteString("\\begin{enumerate}\n")
}

func (exp *Exporter) BeginHeader(macro string, title string, numbered bool, renderedTitle string) {
	ctx := exp.Context()
	cmd := latexHeaderName(macro)
	if !numbered {
		cmd += "*"
	}
	fmt.Fprintf(ctx.GetW(), "\\%s{", cmd)
}

func (exp *Exporter) BeginItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\\item ")
}

func (exp *Exporter) BeginEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\\item ")
}

func (exp *Exporter) BeginItemList() {
	exp.Context().W.WriteString("\\begin{itemize}\n")
}

func (exp *Exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{", id)
	}
	mtag, okMtag := ctx.Mtags[tag]
	if !okMtag {
		fmt.Fprint(w, "\\emph{")
	} else {
		if mtag.Cmd != "" {
			fmt.Fprintf(w, "\\%s{", mtag.Cmd)
		}
	}
	if okMtag {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *Exporter) BeginParagraph() {
	// nothing to do
}

func (exp *Exporter) BeginTable(title string, count int, ncols int) {
	w := exp.Context().GetW()
	if title != "" {
		fmt.Fprint(w, "\\begin{table}[htbp]\n")
	}
	lll := strings.Repeat("l", ncols)
	fmt.Fprintf(w, "\\begin{tabular}{%s}\n", lll)
}

func (exp *Exporter) BeginTableCell() {
	ctx := exp.Context()
	if ctx.TableCell > 1 {
		fmt.Fprint(ctx.GetW(), " & ")
	}
}

func (exp *Exporter) BeginTableRow() {
	// nothing to do
}

func (exp *Exporter) BeginVerse(title string, count int) {
	w := exp.Context().GetW()
	if title != "" {
		fmt.Fprintf(w, "\\poemtitle{%s}\n", title)
		fmt.Fprintf(w, "\\label{poem:%d}\n", count)
	}
	fmt.Fprint(w, "\\begin{verse}\n")
}

func (exp *Exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing to be done for now
}

func (exp *Exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *Exporter) CrossReference(id string, name string, loXentry *frundis.LoXinfo, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	if loXentry != nil {
		fmt.Fprintf(w, "\\hyperref[%s:%d]{", loXentry.Ref, loXentry.Count)
	} else if id != "" {
		ref, _ := ctx.IDs[id] // we know that it's ok
		fmt.Fprintf(w, "\\hyperlink{%s}{", ref)
	}
	fmt.Fprintf(w, "%s}%s", name, punct)
}

func (exp *Exporter) DescName(name string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, "\\item[%s] ", name)
}

func (exp *Exporter) EndDescList() {
	exp.Context().W.WriteString("\\end{description}\n")
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
			fmt.Fprintf(w, "\\end{%s}\n", cmd)
		}
	}
}

func (exp *Exporter) EndEnumList() {
	exp.Context().W.WriteString("\\end{enumerate}\n")
}

func (exp *Exporter) EndEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndHeader(macro string, title string, numbered bool, titleText string) {
	// TODO: rethink args (pass loxinfo?)
	ctx := exp.Context()
	w := ctx.GetW()
	cmd := latexHeaderName(macro)
	fmt.Fprint(w, "}\n")
	if !numbered {
		fmt.Fprintf(w, "\\addcontentsline{toc}{%s}{%s}\n", cmd, titleText)
	}
	toc, _ := ctx.LoXInfo["toc"]
	entry, _ := toc[title]
	fmt.Fprintf(w, "\\label{s:%d}\n", entry.Count)
}

func (exp *Exporter) EndItemList() {
	exp.Context().W.WriteString("\\end{itemize}\n")
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
	if !okMtag {
		fmt.Fprint(w, "}")
	} else {
		if mtag.Cmd != "" {
			fmt.Fprint(w, "}")
		}
	}
	if id != "" {
		fmt.Fprint(w, "}")
	}
	fmt.Fprint(w, punct)
}

func (exp *Exporter) EndParagraph() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n\n")
}

func (exp *Exporter) EndParagraphSoftly() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndParagraphUnsoftly() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *Exporter) EndTable(tableinfo *frundis.TableInfo) {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "\\end{tabular}\n")
	if tableinfo != nil {
		fmt.Fprintf(w, "\\caption{%s}\n", tableinfo.Title)
		fmt.Fprintf(w, "\\label{tbl:%d}\n", ctx.TableCount)
		fmt.Fprint(w, "\\end{table}\n")
	}
}

func (exp *Exporter) EndTableCell() {
}

func (exp *Exporter) EndTableRow() {
	w := exp.Context().GetW()
	fmt.Fprint(w, " \\\\\n")
}

func (exp *Exporter) EndVerse() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\\end{verse}\n")
}

func (exp *Exporter) EndVerseLine() {
	w := exp.Context().GetW()
	fmt.Fprint(w, " \\\\\n")
}

func (exp *Exporter) FormatParagraph(text []byte) []byte {
	return text
}

func (exp *Exporter) FigureImage(image string, label string, link string) {
	bctx := exp.BaseContext()
	if strings.ContainsAny(image, "{}") || strings.ContainsAny(label, "{}") {
		bctx.Error("path argument and label should not contain the characters `{', or `}")
		return
	}
	ctx := exp.Context()
	w := ctx.GetW()
	_, err := os.Stat(image)
	if err != nil {
		bctx.Error("image not found:", image)
		return
	}
	image = escape.EscapeLatexPercent(image)
	fmt.Fprint(w, "\\begin{center}\n")
	fmt.Fprint(w, "\\begin{figure}[htbp]\n")
	fmt.Fprintf(w, "\\includegraphics{%s}\n", image)
	fmt.Fprintf(w, "\\caption{%s}\n", label)
	fmt.Fprintf(w, "\\label{fig:%d}\n", ctx.FigCount)
	fmt.Fprint(w, "\\end{figure}\n")
	fmt.Fprint(w, "\\end{center}\n")
}

func (exp *Exporter) GenRef(prefix string, id string, hasfile bool) string {
	if prefix == "" {
		return fmt.Sprintf("%s", id)
	} else {
		return fmt.Sprintf("%s", prefix)
	}
}

func (exp *Exporter) HeaderReference(macro string) string {
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
	image = escape.EscapeLatexPercent(image)
	fmt.Fprintf(w, "\\includegraphics{%s}%s", image, punct)
}

func (exp *Exporter) LkWithLabel(uri string, label string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.EscapeLatexPercent(parsedURL.String())
	}
	fmt.Fprintf(w, "\\href{%s}{%s}%s", u, escape.EscapeLatexString(label), punct)
}

func (exp *Exporter) LkWithoutLabel(uri string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.EscapeLatexPercent(parsedURL.String())
	}
	fmt.Fprintf(w, "\\url{%s}%s", u, punct)
}

func (exp *Exporter) ParagraphTitle(title string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, "\\paragraph{%s}\n", title)
}

func (exp *Exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return escape.EscapeLatexString(exp.BaseContext().InlinesToText(text))
}

func (exp *Exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	w := exp.Context().GetW()
	bctx := exp.BaseContext()
	if flags["summary"] {
		fmt.Fprint(w, "\\setcounter{tocdepth}{0}\n")
	} else {
		fmt.Fprint(w, "\\setcounter{tocdepth}{3}\n")
	}
	if flags["mini"] {
		switch {
		case flags["lof"]:
			fmt.Fprint(w, "\\minilof\n")
		case flags["lot"]:
			fmt.Fprint(w, "\\minilot\n")
		default:
			fmt.Fprint(w, "\\minitoc\n")
		}
	} else {
		switch {
		case flags["lof"]:
			fmt.Fprint(w, "\\listoffigures\n")
		case flags["lot"]:
			fmt.Fprint(w, "\\listoftables\n")
		case flags["lop"]:
			// XXX: do something about this?
			bctx.Error("list of poems not available for LaTeX")
		default:
			fmt.Fprint(w, "\\tableofcontents\n")
		}
	}
}

func (exp *Exporter) TableOfContentsInfos(flags map[string]bool) {
	if !flags["mini"] {
		return
	}
	exp.minitoc = true
	if flags["lof"] {
		exp.dominilof = true
	}
	if flags["lot"] {
		exp.dominilot = true
	}
	if flags["toc"] {
		exp.dominitoc = true
	}
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
		c = "emph"
	} else {
		c = *cmd
	}
	return frundis.Mtag{Begin: escape.EscapeLatexString(begin), End: escape.EscapeLatexString(end), Cmd: c}
}
