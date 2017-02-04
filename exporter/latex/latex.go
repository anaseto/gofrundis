package latex

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/escape"
	"github.com/anaseto/gofrundis/frundis"
)

type Options struct {
	OutputFile string // name of output file or directory
	Standalone bool   // generate complete document with headers
}

// NewExporter returns a frundis.Exporter suitable to produce LaTeX.
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
}

func (exp *exporter) Init() {
	bctx := &frundis.BaseContext{Format: "latex"}
	exp.Bctx = bctx
	bctx.Init()
	ctx := &frundis.Context{W: bufio.NewWriter(os.Stdout)}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.LaTeX
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
		exp.beginLatexDocument()
	}
	return nil
}

func (exp *exporter) PostProcessing() {
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

func (exp *exporter) BaseContext() *frundis.BaseContext {
	return exp.Bctx
}

func (exp *exporter) BlockHandler() {
	frundis.BlockHandler(exp)
}

func (exp *exporter) BeginDescList() {
	exp.Context().W.WriteString("\\begin{description}\n")
}

func (exp *exporter) BeginDescValue() {
	// do nothing
}

func (exp *exporter) BeginDialogue() {
	ctx := exp.Context()
	w := ctx.GetW()
	dmark, ok := ctx.Params["dmark"]
	if !ok {
		dmark = "---"
	} else {
		dmark = escape.LaTeX(dmark)
	}
	fmt.Fprint(w, dmark)
}

func (exp *exporter) BeginDisplayBlock(tag string, id string) {
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

func (exp *exporter) BeginEnumList() {
	exp.Context().W.WriteString("\\begin{enumerate}\n")
}

func (exp *exporter) BeginHeader(macro string, title string, numbered bool, renderedTitle string) {
	ctx := exp.Context()
	cmd := latexHeaderName(macro)
	if !numbered {
		cmd += "*"
	}
	fmt.Fprintf(ctx.GetW(), "\\%s{", cmd)
}

func (exp *exporter) BeginItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\\item ")
}

func (exp *exporter) BeginEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\\item ")
}

func (exp *exporter) BeginItemList() {
	exp.Context().W.WriteString("\\begin{itemize}\n")
}

func (exp *exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.GetW()
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{", id)
	}
	mtag, ok := ctx.Mtags[tag]
	if !ok {
		fmt.Fprint(w, "\\emph")
	} else {
		fmt.Fprintf(w, "\\%s", mtag.Cmd)
	}
	pairs := mtag.Pairs
	if len(pairs) > 0 {
		fmt.Fprint(w, "[")
		for i := 0; i < len(pairs)-1; i += 2 {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, "%s=%s", escape.LaTeX(pairs[i]), escape.LaTeX(pairs[i+1]))
		}
		fmt.Fprint(w, "]")
	}
	fmt.Fprint(w, "{")
	if ok {
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
	if title != "" {
		fmt.Fprint(w, "\\begin{table}[htbp]\n")
	}
	lll := strings.Repeat("l", ncols)
	fmt.Fprintf(w, "\\begin{tabular}{%s}\n", lll)
}

func (exp *exporter) BeginTableCell() {
	ctx := exp.Context()
	if ctx.TableCell > 1 {
		fmt.Fprint(ctx.GetW(), " & ")
	}
}

func (exp *exporter) BeginTableRow() {
	// nothing to do
}

func (exp *exporter) BeginVerse(title string, count int) {
	w := exp.Context().GetW()
	if title != "" {
		fmt.Fprintf(w, "\\poemtitle{%s}\n", title)
		fmt.Fprintf(w, "\\label{poem:%d}\n", count)
	}
	fmt.Fprint(w, "\\begin{verse}\n")
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
		fmt.Fprintf(w, "\\hyperref[%s:%d]{%s}%s", loXentry.Ref, loXentry.Count, name, punct)
	case id != "":
		ref, _ := ctx.IDs[id] // we know that it's ok
		fmt.Fprintf(w, "\\hyperlink{%s}{%s}%s", ref, name, punct)
	default:
		fmt.Fprintf(w, "%s%s", name, punct)
	}
}

func (exp *exporter) DescName(name string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, "\\item[%s] ", name)
}

func (exp *exporter) EndDescList() {
	exp.Context().W.WriteString("\\end{description}\n")
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
			fmt.Fprintf(w, "\\end{%s}\n", cmd)
		}
	}
}

func (exp *exporter) EndEnumList() {
	exp.Context().W.WriteString("\\end{enumerate}\n")
}

func (exp *exporter) EndEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndHeader(macro string, title string, numbered bool, titleText string) {
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

func (exp *exporter) EndItemList() {
	exp.Context().W.WriteString("\\end{itemize}\n")
}

func (exp *exporter) EndItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	mtag, ok := ctx.Mtags[tag]
	if ok {
		fmt.Fprint(w, mtag.End)
	}
	fmt.Fprint(w, "}")
	if id != "" {
		fmt.Fprint(w, "}")
	}
	fmt.Fprint(w, punct)
}

func (exp *exporter) EndParagraph() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n\n")
}

func (exp *exporter) EndParagraphSoftly() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndParagraphUnsoftly() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndTable(tableinfo *frundis.TableInfo) {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "\\end{tabular}\n")
	if tableinfo != nil {
		fmt.Fprintf(w, "\\caption{%s}\n", tableinfo.Title)
		fmt.Fprintf(w, "\\label{tbl:%d}\n", ctx.TableCount)
		fmt.Fprint(w, "\\end{table}\n")
	}
}

func (exp *exporter) EndTableCell() {
}

func (exp *exporter) EndTableRow() {
	w := exp.Context().GetW()
	fmt.Fprint(w, " \\\\\n")
}

func (exp *exporter) EndVerse() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "\\end{verse}\n")
}

func (exp *exporter) EndVerseLine() {
	w := exp.Context().GetW()
	fmt.Fprint(w, " \\\\\n")
}

func (exp *exporter) FormatParagraph(text []byte) []byte {
	return text
}

func (exp *exporter) FigureImage(image string, label string, link string) {
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
	image = escape.LaTeXPercent(image)
	fmt.Fprint(w, "\\begin{center}\n")
	fmt.Fprint(w, "\\begin{figure}[htbp]\n")
	fmt.Fprintf(w, "\\includegraphics{%s}\n", image)
	fmt.Fprintf(w, "\\caption{%s}\n", label)
	fmt.Fprintf(w, "\\label{fig:%d}\n", ctx.FigCount)
	fmt.Fprint(w, "\\end{figure}\n")
	fmt.Fprint(w, "\\end{center}\n")
}

func (exp *exporter) GenRef(prefix string, id string, hasfile bool) string {
	if prefix == "" {
		return fmt.Sprintf("%s", id)
	} else {
		return fmt.Sprintf("%s", prefix)
	}
}

func (exp *exporter) HeaderReference(macro string) string {
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
	image = escape.LaTeXPercent(image)
	fmt.Fprintf(w, "\\includegraphics{%s}%s", image, punct)
}

func (exp *exporter) LkWithLabel(uri string, label string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.LaTeXPercent(parsedURL.String())
	}
	fmt.Fprintf(w, "\\href{%s}{%s}%s", u, escape.LaTeX(label), punct)
}

func (exp *exporter) LkWithoutLabel(uri string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = escape.LaTeXPercent(parsedURL.String())
	}
	fmt.Fprintf(w, "\\url{%s}%s", u, punct)
}

func (exp *exporter) ParagraphTitle(title string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, "\\paragraph{%s}\n", title)
}

func (exp *exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return escape.LaTeX(exp.BaseContext().InlinesToText(text))
}

func (exp *exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
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

func (exp *exporter) TableOfContentsInfos(flags map[string]bool) {
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

func (exp *exporter) Xdtag(cmd string) frundis.Dtag {
	return frundis.Dtag{Cmd: cmd}
}

func (exp *exporter) Xftag(shell string) frundis.Ftag {
	return frundis.Ftag{Shell: shell}
}

func (exp *exporter) Xmtag(cmd *string, begin string, end string, pairs []string) frundis.Mtag {
	var c string
	if cmd == nil {
		c = "emph"
	} else {
		c = *cmd
	}
	// TODO: perhaps process pairs here and do some error checking
	return frundis.Mtag{Begin: escape.LaTeX(begin), End: escape.LaTeX(end), Cmd: c, Pairs: pairs}
}
