package latex

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/escape"
	"github.com/anaseto/gofrundis/frundis"
)

// Options gathers configuration for LaTeX exporter.
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
	ctx := &frundis.Context{Wout: bufio.NewWriter(os.Stdout), Format: "latex"}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.LaTeX
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
	if exp.Standalone {
		exp.beginLatexDocument()
	}
	return nil
}

func (exp *exporter) PostProcessing() {
	ctx := exp.Context()
	if exp.Standalone {
		exp.EndLatexDocument()
	}
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
	w := exp.Context().W()
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}", id)
	}
	fmt.Fprint(w, "\\begin{description}\n")
}

func (exp *exporter) BeginDescValue() {
	// do nothing
}

func (exp *exporter) BeginDialogue() {
	ctx := exp.Context()
	w := ctx.W()
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
	w := ctx.W()
	dtag, ok := ctx.Dtags[tag]
	var cmd string
	if ok {
		cmd = dtag.Cmd
	}
	if cmd != "" {
		fmt.Fprintf(w, "\\begin{%s}", cmd)
		pairs := dtag.Pairs
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
		fmt.Fprint(w, "\n")
	}
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}\n", id)
	}
}

func (exp *exporter) BeginEnumList(id string) {
	w := exp.Context().W()
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}", id)
	}
	fmt.Fprint(w, "\\begin{enumerate}\n")
}

func (exp *exporter) BeginHeader(macro string, numbered bool, title string) {
	ctx := exp.Context()
	cmd := latexHeaderName(macro)
	if !numbered {
		cmd += "*"
	}
	fmt.Fprintf(ctx.W(), "\\%s{", cmd)
}

func (exp *exporter) BeginItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\\item ")
}

func (exp *exporter) BeginEnumItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\\item ")
}

func (exp *exporter) BeginItemList(id string) {
	w := exp.Context().W()
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}", id)
	}
	fmt.Fprint(w, "\\begin{itemize}\n")
}

func (exp *exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.W()
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}", id)
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

func (exp *exporter) BeginTable(tableinfo *frundis.TableData) {
	w := exp.Context().W()
	if tableinfo.Title != "" {
		fmt.Fprint(w, "\\begin{table}[htbp]\n")
	}
	lll := strings.Repeat("l", tableinfo.Cols)
	fmt.Fprintf(w, "\\begin{tabular}{%s}\n", lll)
}

func (exp *exporter) BeginTableCell() {
	ctx := exp.Context()
	if ctx.Table.Cell > 1 {
		fmt.Fprint(ctx.W(), " & ")
	}
}

func (exp *exporter) BeginTableRow() {
	// nothing to do
}

func (exp *exporter) BeginVerse(title string, id string) {
	w := exp.Context().W()
	if title != "" {
		fmt.Fprintf(w, "\\poemtitle{%s}\n", title)
		fmt.Fprintf(w, "\\label{poem:%s}\n", id)
	} else if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}", id)
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

func (exp *exporter) CrossReference(idf frundis.IDInfo, name string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	switch idf.Type {
	case frundis.HeaderID, frundis.FigureID, frundis.TableID, frundis.PoemID:
		fmt.Fprintf(w, "\\hyperref[%s]{%s}%s", idf.Ref, name, punct)
	case frundis.NoID:
		fmt.Fprintf(w, "%s%s", name, punct)
	default:
		fmt.Fprintf(w, "\\hyperlink{%s}{%s}%s", idf.Ref, name, punct)
	}
}

func (exp *exporter) DescName(name string) {
	w := exp.Context().W()
	fmt.Fprintf(w, "\\item[%s] ", name)
}

func (exp *exporter) EndDescList() {
	exp.Context().Wout.WriteString("\\end{description}\n")
}

func (exp *exporter) EndDescValue() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndDisplayBlock(tag string) {
	if tag != "" {
		ctx := exp.Context()
		w := ctx.W()
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
	exp.Context().Wout.WriteString("\\end{enumerate}\n")
}

func (exp *exporter) EndEnumItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndHeader(macro string, numbered bool, title string) {
	ctx := exp.Context()
	w := ctx.W()
	cmd := latexHeaderName(macro)
	fmt.Fprint(w, "}\n")
	if !numbered {
		fmt.Fprintf(w, "\\addcontentsline{toc}{%s}{%s}\n", cmd, title)
	}
	fmt.Fprintf(w, "\\label{s:%d}\n", ctx.Toc.HeaderCount)
}

func (exp *exporter) EndItemList() {
	exp.Context().Wout.WriteString("\\end{itemize}\n")
}

func (exp *exporter) EndItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	mtag, ok := ctx.Mtags[tag]
	if ok {
		fmt.Fprint(w, mtag.End)
	}
	fmt.Fprint(w, "}")
	fmt.Fprint(w, punct)
}

func (exp *exporter) EndParagraph() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n\n")
}

func (exp *exporter) EndParagraphSoftly() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndParagraphUnsoftly() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndTable(tableinfo *frundis.TableData) {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "\\end{tabular}\n")
	if tableinfo.Title != "" {
		fmt.Fprintf(w, "\\caption{%s}\n", tableinfo.Title)
		fmt.Fprintf(w, "\\label{tbl:%d}\n", ctx.Table.TitCount)
		fmt.Fprint(w, "\\end{table}\n")
	} else if tableinfo.ID != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}", tableinfo.ID)
	}
}

func (exp *exporter) EndTableCell() {
}

func (exp *exporter) EndTableRow() {
	w := exp.Context().W()
	fmt.Fprint(w, " \\\\\n")
}

func (exp *exporter) EndVerse() {
	w := exp.Context().W()
	fmt.Fprint(w, "\\end{verse}\n")
}

func (exp *exporter) EndVerseLine() {
	w := exp.Context().W()
	fmt.Fprint(w, " \\\\\n")
}

func (exp *exporter) FormatParagraph(text []byte) []byte {
	return text
}

func (exp *exporter) FigureImage(image string, label string, link string) {
	ctx := exp.Context()
	if strings.ContainsAny(image, "{}") || strings.ContainsAny(label, "{}") {
		ctx.Error("path argument and label should not contain the characters `{', or `}")
		return
	}
	w := ctx.W()
	_, err := os.Stat(image)
	if err != nil {
		ctx.Error("image not found:", image)
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
	if prefix != "" {
		return fmt.Sprintf("%s:%s", prefix, id)
	} else {
		return fmt.Sprintf("%s", id)
	}
}

func (exp *exporter) HeaderReference(macro string) string {
	return exp.GenRef("s", strconv.Itoa(exp.Context().Toc.HeaderCount), false)
}

func (exp *exporter) InlineImage(image string, link string, id string, punct string) {
	ctx := exp.Context()
	if strings.ContainsAny(image, "{}") {
		ctx.Error("path argument and label should not contain the characters `{', or `}")
		return
	}
	w := ctx.W()
	_, err := os.Stat(image)
	if err != nil {
		ctx.Error("image not found:", image)
		return
	}
	image = escape.LaTeXPercent(image)
	fmt.Fprintf(w, "\\includegraphics{%s}%s", image, punct)
	if id != "" {
		fmt.Fprintf(w, "\\hypertarget{%s}{}", id)
	}
}

func (exp *exporter) LkWithLabel(uri string, label string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		ctx.Error("invalid url or path:", uri)
	} else {
		u = escape.LaTeXPercent(parsedURL.String())
	}
	fmt.Fprintf(w, "\\href{%s}{%s}%s", u, escape.LaTeX(label), punct)
}

func (exp *exporter) LkWithoutLabel(uri string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		ctx.Error("invalid url or path:", uri)
	} else {
		u = escape.LaTeXPercent(parsedURL.String())
	}
	fmt.Fprintf(w, "\\url{%s}%s", u, punct)
}

func (exp *exporter) ParagraphTitle(title string) {
	w := exp.Context().W()
	fmt.Fprintf(w, "\\paragraph{%s}\n", title)
}

func (exp *exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return escape.LaTeX(exp.Context().InlinesToText(text))
}

func (exp *exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	ctx := exp.Context()
	w := ctx.W()
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
			ctx.Error("list of poems not available for LaTeX")
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

func (exp *exporter) Xdtag(cmd string, pairs []string) frundis.Dtag {
	return frundis.Dtag{Cmd: cmd, Pairs: pairs}
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
