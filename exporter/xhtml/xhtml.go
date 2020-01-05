package xhtml

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/frundis"
)

// Options gathers configuration for HTML and EPUB export.
type Options struct {
	AllInOneFile bool   // output goes only to one html file
	Format       string // "epub" or "xhtml"
	OutputFile   string // name of output file or directory
	Standalone   bool   // generate complete document with headers (default unless AllInOneFile)
	Werror       io.Writer
}

// NewExporter returns a frundis.Exporter suitable to produce EPUB or HTML.
// See type Options for options.
func NewExporter(opts *Options) frundis.Exporter {
	return &exporter{
		AllInOneFile: opts.AllInOneFile,
		Format:       opts.Format,
		OutputFile:   opts.OutputFile,
		Standalone:   opts.Standalone,
		Werror:       opts.Werror}
}

type exporter struct {
	Ctx                 *frundis.Context
	Format              string // "epub" or "xhtml"
	AllInOneFile        bool
	Standalone          bool
	OutputFile          string
	Werror              io.Writer
	curOutputFile       *os.File
	xhtmlNavigationText *bytes.Buffer
}

func (exp *exporter) Init() {
	ctx := &frundis.Context{Wout: bufio.NewWriter(os.Stdout), Format: exp.Format}
	exp.Ctx = ctx
	ctx.Werror = exp.Werror
	ctx.Init()
	ctx.Params["xhtml-index"] = "full"
	ctx.Filters["escape"] = escapeFilter
	exp.xhtmlNavigationText = &bytes.Buffer{}
}

func (exp *exporter) Reset() error {
	ctx := exp.Context()
	ctx.Reset()
	switch exp.Format {
	case "xhtml":
		if exp.OutputFile != "" && !exp.AllInOneFile {
			_, err := os.Stat(exp.OutputFile)
			if err != nil {
				err = os.Mkdir(exp.OutputFile, 0755)
				if err != nil {
					return fmt.Errorf("%v\n", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "warning: directory %s already exists\n", exp.OutputFile)
			}
			index := path.Join(exp.OutputFile, "index.html")
			exp.curOutputFile, err = os.Create(index)
			if err != nil {
				return fmt.Errorf("%v\n", err)
			}
		} else if exp.OutputFile != "" {
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
		if exp.Standalone || !exp.AllInOneFile {
			exp.XHTMLdocumentHeader(ctx.Wout, ctx.Params["document-title"])
			exp.xhtmlTitlePage()
			if !exp.AllInOneFile {
				switch ctx.Params["xhtml-index"] {
				case "full":
					opts := make(map[string][]ast.Inline)
					flags := make(map[string]bool)
					exp.writeTOC(ctx.Wout, xhtmlToc, opts, flags)
				case "summary":
					opts := make(map[string][]ast.Inline)
					flags := map[string]bool{"summary": true}
					exp.writeTOC(ctx.Wout, xhtmlToc, opts, flags)
				}
			}
		}
	case "epub":
		err := makeDirectory(exp.OutputFile)
		if err != nil {
			return err
		}
		epub := path.Join(exp.OutputFile, "EPUB")
		err = makeDirectory(epub)
		if err != nil {
			return err
		}
		metainf := path.Join(exp.OutputFile, "META-INF")
		err = makeDirectory(metainf)
		if err != nil {
			return err
		}
		exp.epubGen()

		exp.curOutputFile, err = os.Create(path.Join(exp.OutputFile, "EPUB", "index.xhtml"))
		if err != nil {
			return fmt.Errorf("%v\n", err)
		}
		ctx.Wout = bufio.NewWriter(exp.curOutputFile)
		exp.XHTMLdocumentHeader(ctx.Wout, ctx.Params["document-title"])
		exp.xhtmlTitlePage()
	}
	return nil
}

func makeDirectory(filename string) error {
	_, err := os.Stat(filename)
	if err != nil {
		err = os.Mkdir(filename, 0755)
		if err != nil {
			return fmt.Errorf("%v\n", err)
		}
	}
	return nil
}

func (exp *exporter) PostProcessing() {
	ctx := exp.Context()
	switch ctx.Format {
	case "xhtml":
		if exp.xhtmlNavigationText.Len() > 0 {
			ctx.Wout.Write(exp.xhtmlNavigationText.Bytes())
		}
		if exp.Standalone || !exp.AllInOneFile {
			exp.XHTMLdocumentFooter(ctx.Wout)
		}
	case "epub":
		exp.XHTMLdocumentFooter(ctx.Wout)
	}
	ctx.Wout.Flush()
	if exp.curOutputFile != nil {
		err := exp.curOutputFile.Close()
		if err != nil {
			ctx.Error(err)
		}
	}
}

func (exp *exporter) BeginDescList(id string) {
	ctx := exp.Context()
	w := ctx.W()
	if id != "" {
		id = " id=\"" + id + "\""
	}
	fmt.Fprintf(w, "<dl%s>\n", id)
}

func (exp *exporter) BeginDescValue() {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "<dd>")
}

func (exp *exporter) BeginDialogue() {
	ctx := exp.Context()
	dmark, ok := ctx.Params["dmark"]
	if !ok {
		dmark = "–"
	}
	w := ctx.W()
	fmt.Fprint(w, dmark)
}

func (exp *exporter) BeginDisplayBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.W()
	dtag, ok := ctx.Dtags[tag]
	if ok {
		var cmd string
		if ok {
			cmd = dtag.Cmd
		}
		if cmd == "" {
			cmd = "div"
		}
		fmt.Fprintf(w, "<%s class=\"%s\"", cmd, tag)
	} else {
		fmt.Fprint(w, "<div")
	}
	pairs := dtag.Pairs
	for i := 0; i < len(pairs)-1; i += 2 {
		fmt.Fprintf(w, " %s=\"%s\"", html.EscapeString(pairs[i]), html.EscapeString(pairs[i+1]))
	}
	if id != "" {
		fmt.Fprintf(w, " id=\"%s\"", id)
	}
	fmt.Fprint(w, ">\n")
}

func (exp *exporter) BeginEnumList(id string) {
	ctx := exp.Context()
	w := ctx.W()
	if id != "" {
		id = " id=\"" + id + "\""
	}
	fmt.Fprintf(w, "<ol%s>\n", id)
}

func (exp *exporter) BeginHeader(macro string, numbered bool, title string) {
	ctx := exp.Context()
	num := ctx.Toc.HeaderLevel(macro)
	switch macro {
	case "Pt", "Ch":
		if ctx.Format == "epub" || !exp.AllInOneFile {
			exp.xhtmlFileOutputChange(title)
		}
	}
	w := ctx.W()
	toc, _ := ctx.LoXstack["toc"]
	entry := toc[ctx.Toc.HeaderCount-1] // headers count is updated before
	id := exp.getID(entry)
	fmt.Fprintf(w, "<h%d class=\"%s\" id=\"%s\">", num, macro, id)
	if numbered {
		fmt.Fprintf(w, "%s ", entry.Num)
	}
}

func (exp *exporter) BeginItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "<li>")
}

func (exp *exporter) BeginEnumItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "<li>")
}

func (exp *exporter) BeginItemList(id string) {
	w := exp.Context().W()
	if id != "" {
		id = " id=\"" + id + "\""
	}
	fmt.Fprintf(w, "<ul%s>\n", id)
}

func (exp *exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	mtag, ok := ctx.Mtags[tag]
	w := ctx.W()
	if !ok {
		fmt.Fprint(w, "<em")
	} else {
		fmt.Fprintf(w, "<%s class=\"%s\"", mtag.Cmd, tag)
	}
	pairs := mtag.Pairs
	for i := 0; i < len(pairs)-1; i += 2 {
		fmt.Fprintf(w, " %s=\"%s\"", html.EscapeString(pairs[i]), html.EscapeString(pairs[i+1]))
	}
	if id != "" {
		fmt.Fprintf(w, " id=\"%s\"", id)
	}
	fmt.Fprint(w, ">")
	if ok {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *exporter) BeginParagraph() {
	w := exp.Context().W()
	fmt.Fprint(w, "<p>")
}

func (exp *exporter) BeginPhrasingMacroInParagraph(nospace bool) {
	frundis.BeginPhrasingMacroInParagraph(exp, nospace)
}

func (exp *exporter) BeginTable(tableinfo *frundis.TableData) {
	ctx := exp.Context()
	w := ctx.W()
	var id string
	if tableinfo.Title != "" {
		fmt.Fprintf(w, "<div id=\"tbl%d\" class=\"table\">\n", ctx.Table.TitCount)
	} else if tableinfo.ID != "" {
		id = " id=\"" + tableinfo.ID + "\""
	}
	fmt.Fprintf(w, "<table%s>\n", id)
}

func (exp *exporter) BeginTableCell() {
	w := exp.Context().W()
	fmt.Fprint(w, "<td>")
}

func (exp *exporter) BeginTableRow() {
	w := exp.Context().W()
	fmt.Fprint(w, "<tr>\n")
}

func (exp *exporter) BeginVerse(title string, id string) {
	w := exp.Context().W()
	if title != "" {
		fmt.Fprint(w, "<div class=\"verse\">\n")
		fmt.Fprintf(w, "<h4 id=\"poem%s\">%s</h4>\n", id, title)
		return
	}
	if id != "" {
		id = " id=\"" + id + "\""
	}
	fmt.Fprintf(w, "<div class=\"verse\"%s>\n", id)
}

func (exp *exporter) BeginVerseLine() {
	w := exp.Context().W()
	fmt.Fprint(w, "<span class=\"verse\">")
}

func (exp *exporter) CheckParamAssignement(param string, value string) bool {
	ctx := exp.Context()
	switch param {
	case "xhtml-index":
		switch value {
		case "full", "summary", "none":
		default:
			ctx.Error("xhtml-index parameter:unknown value:", value)
			return false
		}
	case "epub-version":
		if value != "2" && value != "3" {
			ctx.Error("epub-version parameter should be 2 or 3 but got ", value)
			return false
		}
	}
	return true
}

func (exp *exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *exporter) CrossReference(idf frundis.IDInfo, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "<a")
	switch idf.Type {
	case frundis.NoID:
	default:
		fmt.Fprintf(w, " href=\"%s\"", idf.Ref)
	}
	fmt.Fprintf(w, ">%s</a>%s", idf.Name, punct)
}

func (exp *exporter) DescName(name string) {
	w := exp.Context().W()
	fmt.Fprintf(w, "<dt>%s</dt>\n", name)
}

func (exp *exporter) EndDescList() {
	w := exp.Context().W()
	fmt.Fprint(w, "</dl>\n")
}

func (exp *exporter) EndDescValue() {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "</dd>\n")
}

func (exp *exporter) EndDisplayBlock(tag string) {
	ctx := exp.Context()
	w := ctx.W()
	if tag != "" {
		dtag, ok := ctx.Dtags[tag]
		var cmd string
		if ok {
			cmd = dtag.Cmd
		}
		if cmd == "" {
			cmd = "div"
		}
		fmt.Fprintf(w, "</%s>\n", cmd)
	} else {
		fmt.Fprint(w, "</div>\n")
	}
}

func (exp *exporter) EndEnumList() {
	w := exp.Context().W()
	fmt.Fprint(w, "</ol>\n")
}

func (exp *exporter) EndEnumItem() {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "</li>\n")
}

func (exp *exporter) EndHeader(macro string, numbered bool, title string) {
	ctx := exp.Context()
	w := ctx.W()
	num := ctx.Toc.HeaderLevel(macro)
	fmt.Fprintf(w, "</h%d>\n", num)
}

func (exp *exporter) EndItemList() {
	w := exp.Context().W()
	fmt.Fprint(w, "</ul>\n")
}

func (exp *exporter) EndItem() {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "</li>\n")
}

func (exp *exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	mtag, ok := ctx.Mtags[tag]
	if !ok {
		fmt.Fprint(w, "</em>")
	} else {
		fmt.Fprint(w, mtag.End)
		fmt.Fprint(w, fmt.Sprint("</", mtag.Cmd, ">"))
	}
	fmt.Fprint(w, punct)
}

func (exp *exporter) EndParagraph() {
	w := exp.Context().W()
	fmt.Fprint(w, "</p>\n")
}

func (exp *exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *exporter) EndParagraphUnsoftly() {
	// do nothing
}

func (exp *exporter) EndStanza() {
	w := exp.Context().W()
	fmt.Fprint(w, "</span>\n")
	exp.EndParagraph()
}

func (exp *exporter) EndTable(tableinfo *frundis.TableData) {
	w := exp.Context().W()
	fmt.Fprint(w, "</table>\n")
	if tableinfo.Title != "" {
		fmt.Fprintf(w, "<p class=\"table-title\">%s</p>\n</div>\n", tableinfo.Title)
	}
}

func (exp *exporter) EndTableCell() {
	w := exp.Context().W()
	fmt.Fprint(w, "</td>\n")
}

func (exp *exporter) EndTableRow() {
	w := exp.Context().W()
	fmt.Fprint(w, "</tr>\n")
}

func (exp *exporter) EndVerse() {
	w := exp.Context().W()
	fmt.Fprint(w, "</div>\n")
}

func (exp *exporter) EndVerseLine() {
	w := exp.Context().W()
	fmt.Fprint(w, "</span><br />\n")
}

func (exp *exporter) FormatParagraph(text []byte) []byte {
	return text
}

func (exp *exporter) FigureImage(image string, caption string, link string) {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprintf(w, "<div id=\"fig%d\" class=\"figure\">\n", ctx.FigCount)
	if ctx.Format == "epub" {
		image = path.Base(image)
	}
	parsedURL, err := url.Parse(image)
	var u string
	if err != nil {
		ctx.Error("invalid url or path:", image)
	} else {
		u = html.EscapeString(parsedURL.String())
	}
	if ctx.Format == "epub" {
		u = path.Join("images", u)
	}
	link = exp.processLink(link)
	if link != "" && ctx.Format == "xhtml" {
		fmt.Fprintf(w, "  <a href=\"%s\"><img src=\"%s\" alt=\"%s\" /></a>\n", link, u, caption)
	} else {
		fmt.Fprintf(w, "  <img src=\"%s\" alt=\"%s\" />\n", u, caption)
	}
	if caption != "" {
		fmt.Fprintf(w, "  <p class=\"caption\">%s</p>\n", caption)
	}
	fmt.Fprint(w, "</div>\n")
}

func (exp *exporter) GenRef(prefix string, id string, hasfile bool) string {
	ctx := exp.Context()
	toc := ctx.Toc
	var href string
	switch {
	case exp.AllInOneFile:
		href = fmt.Sprintf("#%s%s", prefix, id)
	default:
		var suffix string
		if ctx.Format == "epub" {
			suffix = ".xhtml"
		} else {
			suffix = ".html"
		}
		if hasfile {
			href = fmt.Sprintf("body-%d-%d%s", toc.PartCount, toc.ChapterCount, suffix)
		} else if toc.PartCount > 0 || toc.ChapterCount > 0 {
			href = fmt.Sprintf("body-%d-%d%s#%s%s", toc.PartCount, toc.ChapterCount, suffix, prefix, id)
		} else {
			href = fmt.Sprintf("index%s#%s%s", suffix, prefix, id)
		}
	}
	return href
}

func (exp *exporter) HeaderReference(macro string) string {
	ctx := exp.Context()
	toc := ctx.Toc
	var href string
	switch macro {
	case "Pt", "Ch":
		href = exp.GenRef("s", strconv.Itoa(toc.HeaderCount), true)
	case "Sh", "Ss":
		if exp.AllInOneFile {
			href = exp.GenRef("s", strconv.Itoa(toc.HeaderCount), false)
		} else {
			href = exp.GenRef("s", fmt.Sprintf("%d-%d", toc.SectionCount, toc.SubsectionCount), false)
		}
	}
	return href
}

func (exp *exporter) processLink(link string) string {
	ctx := exp.Context()
	if link == "" {
		return link
	}
	parsedURL, err := url.Parse(link)
	if err != nil {
		ctx.Error("invalid url or path:", link)
		link = ""
	} else {
		link = html.EscapeString(parsedURL.String())
	}
	return link
}

func (exp *exporter) InlineImage(image string, link string, id string, punct string) {
	ctx := exp.Context()
	w := exp.Context().W()
	if ctx.Format == "epub" {
		image = path.Base(image)
	}
	parsedURL, err := url.Parse(image)
	var u string
	if err != nil {
		ctx.Error("invalid url or path:", image)
	} else {
		u = html.EscapeString(parsedURL.String())
	}
	if ctx.Format == "epub" {
		u = path.Join("images", u)
	}
	link = exp.processLink(link)
	if id != "" {
		id = " id=\"" + id + "\""
	}
	if link != "" && ctx.Format == "xhtml" {
		fmt.Fprintf(w, "<a href=\"%s\"><img src=\"%s\" alt=\"\"%s /></a>%s", link, u, id, punct)
	} else {
		fmt.Fprintf(w, "<img src=\"%s\" alt=\"\"%s />%s", u, id, punct)
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
		u = html.EscapeString(parsedURL.String())
	}
	fmt.Fprintf(w, "<a href=\"%s\">%s</a>%s", u, label, punct)
}

func (exp *exporter) LkWithoutLabel(uri string, punct string) {
	exp.LkWithLabel(uri, html.EscapeString(uri), punct)
}

func (exp *exporter) ParagraphTitle(title string) {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprintf(w, "<p class=\"paragraph\"><strong class=\"paragraph\">%s</strong>\n", title)
}

func (exp *exporter) RenderText(text []ast.Inline) string {
	ctx := exp.Context()
	if ctx.Params["lang"] == "fr" {
		text = frundis.FrenchTipography(exp, text)
	}
	return html.EscapeString(ctx.InlinesToText(text))
}

func (exp *exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	w := exp.Context().W()
	switch {
	case flags["toc"]:
		exp.writeTOC(w, xhtmlToc, opts, flags)
	case flags["lot"]:
		exp.xhtmlLoX(w, "lot")
	case flags["lof"]:
		exp.xhtmlLoX(w, "lof")
	case flags["lop"]:
		exp.xhtmlLoX(w, "lop")
	}
}

func (exp *exporter) TableOfContentsInfos(flags map[string]bool) {
	// (dominitoc, etc.) (useless for html)
}

func (exp *exporter) Xdtag(cmd string, pairs []string) frundis.Dtag {
	switch cmd {
	case "address", "article", "aside", "blockquote", "div", "header", "fieldset",
		"figure", "footer", "form", "main", "nav", "section", "":
	default:
		exp.Context().Error(cmd, ":expected element allowing flowing content")
	}
	return frundis.Dtag{Cmd: cmd, Pairs: pairs}
}

func (exp *exporter) Xmtag(cmd *string, begin string, end string, pairs []string) frundis.Mtag {
	var c string
	if cmd == nil || *cmd == "" {
		c = "em"
	} else {
		c = *cmd
	}
	switch c {
	case "a", "abbr", "area", "audio", "b", "bdi", "bdo", "br", "button", "canvas", "cite", "code", "data", "datalist", "del",
		"dfn", "em", "embed", "i", "iframe", "img", "input", "ins", "kbd", "keygen", "label", "link", "map", "mark", "math",
		"meta", "meter", "noscript", "object", "output", "progress", "q", "ruby", "s", "samp", "script", "select",
		"small", "span", "strong", "sub", "sup", "svg", "template", "textarea", "time", "u", "var", "video", "wbr", "text":
	default:
		exp.Context().Errorf("%s: not an html phrasing element", c)
	}
	// TODO: perhaps process pairs here and do some error checking
	return frundis.Mtag{Begin: begin, End: end, Cmd: c, Pairs: pairs}
}
