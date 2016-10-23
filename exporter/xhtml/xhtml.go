package xhtml

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/frundis"
)

type Exporter struct {
	Bctx                *frundis.BaseContext
	Ctx                 *frundis.Context
	Format              string // "epub" or "xhtml"
	AllInOneFile        bool
	Standalone          bool
	OutputFile          string
	curOutputFile       *os.File
	xhtmlNavigationText *bytes.Buffer
}

func (exp *Exporter) Init() {
	bctx := &frundis.BaseContext{Format: exp.Format}
	exp.Bctx = bctx
	bctx.Init()
	ctx := &frundis.Context{W: bufio.NewWriter(os.Stdout)}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Params["xhtml-index"] = "full"
	ctx.Filters["escape"] = escapeFilter
	exp.xhtmlNavigationText = &bytes.Buffer{}
}

func (exp *Exporter) Reset() {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	ctx.Reset()
	bctx.Reset()
	switch exp.Format {
	case "xhtml":
		if exp.OutputFile != "" && !exp.AllInOneFile {
			_, err := os.Stat(exp.OutputFile)
			if err != nil {
				err = os.Mkdir(exp.OutputFile, 0755)
				if err != nil {
					fmt.Fprintf(os.Stderr, "frundis:%v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "frundis:warning:directory %s already exists", exp.OutputFile)
			}
			index := path.Join(exp.OutputFile, "index.html")
			exp.curOutputFile, err = os.Create(index)
			if err != nil {
				fmt.Fprintf(os.Stderr, "frundis:%v\n", err)
				os.Exit(1)
			}
		} else if exp.OutputFile != "" {
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
		if exp.Standalone || !exp.AllInOneFile {
			title := html.EscapeString(ctx.Params["document-title"])
			exp.XHTMLdocumentHeader(ctx.W, title)
			exp.xhtmlTitlePage()
			if !exp.AllInOneFile {
				switch ctx.Params["xhtml-index"] {
				case "full":
					opts := make(map[string][]ast.Inline)
					flags := make(map[string]bool)
					exp.writeTOC(ctx.W, xhtmlToc, opts, flags)
				case "summary":
					opts := make(map[string][]ast.Inline)
					flags := map[string]bool{"summary": true}
					exp.writeTOC(ctx.W, xhtmlToc, opts, flags)
				}
			}
		}
	case "epub":
		makeDirectory(exp.OutputFile)
		epub := path.Join(exp.OutputFile, "EPUB")
		makeDirectory(epub)
		metainf := path.Join(exp.OutputFile, "META-INF")
		makeDirectory(metainf)
		exp.epubGen()

		var err error
		exp.curOutputFile, err = os.Create(path.Join(exp.OutputFile, "EPUB", "index.xhtml"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "frundis:%v\n", err)
			os.Exit(1)
		}
		ctx.W = bufio.NewWriter(exp.curOutputFile)
		title := ctx.Params["document-title"]
		exp.XHTMLdocumentHeader(ctx.W, title)
		exp.xhtmlTitlePage()
	}
}

func makeDirectory(filename string) {
	_, err := os.Stat(filename)
	if err != nil {
		err = os.Mkdir(filename, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "frundis:%v\n", err)
			os.Exit(1)
		}
	}
}

func (exp *Exporter) PostProcessing() {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	switch bctx.Format {
	case "xhtml":
		if exp.xhtmlNavigationText.Len() > 0 {
			ctx.W.Write(exp.xhtmlNavigationText.Bytes())
		}
		if exp.Standalone || !exp.AllInOneFile {
			exp.XHTMLdocumentFooter(ctx.W)
		}
	case "epub":
		exp.XHTMLdocumentFooter(ctx.W)
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
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "<dl>\n")
}

func (exp *Exporter) BeginDescValue() {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "<dd>")
}

func (exp *Exporter) BeginDialogue() {
	ctx := exp.Context()
	dmark, ok := ctx.Params["dmark"]
	if !ok {
		dmark = "–"
	}
	w := ctx.GetW()
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
		if cmd == "" {
			cmd = "div"
		}
		fmt.Fprintf(w, "<%s class=\"%s\"", cmd, tag)
	} else {
		fmt.Fprint(w, "<div")
	}
	if id != "" {
		fmt.Fprintf(w, " id=\"%s\"", id)
	}
	fmt.Fprint(w, ">\n")
}

func (exp *Exporter) BeginEnumList() {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "<ol>\n")
}

func (exp *Exporter) BeginHeader(macro string, title string, numbered bool, renderedTitle string) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	num := ctx.TocInfo.HeaderLevel(macro)
	switch macro {
	case "Pt", "Ch":
		if bctx.Format == "epub" || !exp.AllInOneFile {
			exp.xhtmlFileOutputChange(renderedTitle)
		}
	}
	w := ctx.GetW()
	toc, _ := ctx.LoXInfo["toc"] // should be ok
	entry, _ := toc[title]       // should be ok
	id := exp.getID(entry)
	fmt.Fprintf(w, "<h%d class=\"%s\" id=\"%s\">", num, macro, id)
	if numbered {
		fmt.Fprintf(w, "%s ", entry.Num)
	}
}

func (exp *Exporter) BeginItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<li>")
}

func (exp *Exporter) BeginEnumItem() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<li>")
}

func (exp *Exporter) BeginItemList() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<ul>\n")
}

func (exp *Exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	mtag, okMtag := ctx.Mtags[tag]
	w := ctx.GetW()
	if !okMtag {
		fmt.Fprint(w, "<em")
	} else {
		if mtag.Cmd != "" {
			fmt.Fprintf(w, "<%s class=\"%s\"", mtag.Cmd, tag)
		}
	}
	if id != "" {
		fmt.Fprintf(w, " id=\"%s\"", id)
	}
	if !okMtag || mtag.Cmd != "" {
		fmt.Fprint(w, ">")
	}
	if okMtag {
		fmt.Fprint(w, mtag.Begin)
	}
}

func (exp *Exporter) BeginParagraph() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<p>")
}

func (exp *Exporter) BeginTable(title string, count int, ncols int) {
	w := exp.Context().GetW()
	if title != "" {
		fmt.Fprintf(w, "<div id=\"tbl%d\" class=\"table\">\n", count)
	}
	fmt.Fprint(w, "<table>\n")
}

func (exp *Exporter) BeginTableCell() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<td>")
}

func (exp *Exporter) BeginTableRow() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<tr>\n")
}

func (exp *Exporter) BeginVerse(title string, count int) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<div class=\"verse\">\n")
	if title != "" {
		fmt.Fprintf(w, "<h4 id=\"poem%d\">%s</h4>\n", count, title)
	}
}

func (exp *Exporter) CheckParamAssignement(param string, value string) bool {
	bctx := exp.BaseContext()
	switch param {
	case "xhtml-index":
		switch value {
		case "full", "summary", "none":
		default:
			bctx.Error("xhtml-index parameter:unknown value:", value)
			return false
		}
	case "epub-version":
		if value != "2" && value != "3" {
			bctx.Error("epub-version parameter should be 2 or 3 but got ", value)
			return false
		}
	}
	return true
}

func (exp *Exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *Exporter) CrossReference(id string, name string, loXentry *frundis.LoXinfo, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "<a")
	if loXentry != nil {
		fmt.Fprintf(w, " href=\"%s\"", loXentry.Ref)
	} else if id != "" {
		href, _ := ctx.IDs[id] // we know that it's ok
		fmt.Fprintf(w, " href=\"%s\"", href)
	}
	fmt.Fprintf(w, ">%s</a>%s", name, punct)
}

func (exp *Exporter) DescName(name string) {
	w := exp.Context().GetW()
	fmt.Fprintf(w, "<dt>%s</dt>\n", name)
}

func (exp *Exporter) EndDescList() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</dl>\n")
}

func (exp *Exporter) EndDescValue() {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "</dd>\n")
}

func (exp *Exporter) EndDisplayBlock(tag string) {
	ctx := exp.Context()
	w := ctx.GetW()
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

func (exp *Exporter) EndEnumList() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</ol>\n")
}

func (exp *Exporter) EndEnumItem() {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "</li>\n")
}

func (exp *Exporter) EndHeader(macro string, title string, numbered bool, titleText string) {
	ctx := exp.Context()
	w := ctx.GetW()
	num := ctx.TocInfo.HeaderLevel(macro)
	fmt.Fprintf(w, "</h%d>\n", num)
}

func (exp *Exporter) EndItemList() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</ul>\n")
}

func (exp *Exporter) EndItem() {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprint(w, "</li>\n")
}

func (exp *Exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.GetW()
	mtag, ok := ctx.Mtags[tag]
	if !ok {
		fmt.Fprint(w, "</em>")
	} else {
		fmt.Fprint(w, mtag.End)
		if mtag.Cmd != "" {
			fmt.Fprint(w, "</"+mtag.Cmd+">")
		}
	}
	fmt.Fprint(w, punct)
}

func (exp *Exporter) EndParagraph() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</p>\n")
}

func (exp *Exporter) EndParagraphSoftly() {
	exp.EndParagraph()
}

func (exp *Exporter) EndParagraphUnsoftly() {
	// do nothing
}

func (exp *Exporter) EndTable(tableinfo *frundis.TableInfo) {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</table>\n")
	if tableinfo != nil {
		fmt.Fprintf(w, "<p class=\"table-title\">%s</p>\n</div>\n", tableinfo.Title)
	}
}

func (exp *Exporter) EndTableCell() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</td>\n")
}

func (exp *Exporter) EndTableRow() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</tr>\n")
}

func (exp *Exporter) EndVerse() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "</div>\n")
}

func (exp *Exporter) EndVerseLine() {
	w := exp.Context().GetW()
	fmt.Fprint(w, "<br />\n")
}

func (exp *Exporter) FormatParagraph(text []byte) []byte {
	return text
}

func (exp *Exporter) FigureImage(image string, label string, link string) {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprintf(w, "<div id=\"fig%d\" class=\"figure\">\n", ctx.FigCount)
	if bctx.Format == "epub" {
		image = path.Base(image)
	}
	parsedURL, err := url.Parse(image)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", image)
	} else {
		u = html.EscapeString(parsedURL.String())
	}
	if bctx.Format == "epub" {
		u = path.Join("images", u)
	}
	image = html.EscapeString(image)
	link = exp.processLink(link)
	if link != "" && bctx.Format == "xhtml" {
		fmt.Fprintf(w, "  <a href=\"%s\"><img src=\"%s\" alt=\"%s\" /></a>\n", link, u, image)
	} else {
		fmt.Fprintf(w, "  <img src=\"%s\" alt=\"%s\" />\n", u, image)
	}
	fmt.Fprintf(w, "  <p class=\"caption\">%s</p>\n", label)
	fmt.Fprint(w, "</div>\n")
}

func (exp *Exporter) GenRef(prefix string, id string, hasfile bool) string {
	bctx := exp.BaseContext()
	ctx := exp.Context()
	toc := ctx.TocInfo
	var href string
	switch {
	case exp.AllInOneFile:
		href = fmt.Sprintf("#%s%s", prefix, id)
	default:
		var suffix string
		if bctx.Format == "epub" {
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

func (exp *Exporter) HeaderReference(macro string) string {
	ctx := exp.Context()
	toc := ctx.TocInfo
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

func (exp *Exporter) processLink(link string) string {
	bctx := exp.BaseContext()
	if link == "" {
		return link
	}
	parsedURL, err := url.Parse(link)
	if err != nil {
		bctx.Error("invalid url or path:", link)
		link = ""
	} else {
		link = html.EscapeString(parsedURL.String())
	}
	return link
}

func (exp *Exporter) InlineImage(image string, link string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	if bctx.Format == "epub" {
		image = path.Base(image)
	}
	parsedURL, err := url.Parse(image)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", image)
	} else {
		u = html.EscapeString(parsedURL.String())
	}
	if bctx.Format == "epub" {
		u = path.Join("images", u)
	}
	image = html.EscapeString(image)
	link = exp.processLink(link)
	if link != "" && bctx.Format == "xhtml" {
		fmt.Fprintf(w, "<a href=\"%s\"><img src=\"%s\" alt=\"%s\" /></a>%s", link, u, image, punct)
	} else {
		fmt.Fprintf(w, "<img src=\"%s\" alt=\"%s\" />%s", u, image, punct)
	}
}

func (exp *Exporter) LkWithLabel(uri string, label string, punct string) {
	bctx := exp.BaseContext()
	w := exp.Context().GetW()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		bctx.Error("invalid url or path:", uri)
	} else {
		u = html.EscapeString(parsedURL.String())
	}
	fmt.Fprintf(w, "<a href=\"%s\">%s</a>%s", u, label, punct)
}

func (exp *Exporter) LkWithoutLabel(uri string, punct string) {
	exp.LkWithLabel(uri, html.EscapeString(uri), punct)
}

func (exp *Exporter) ParagraphTitle(title string) {
	ctx := exp.Context()
	w := ctx.GetW()
	fmt.Fprintf(w, "<p class=\"paragraph\"><strong class=\"paragraph\">%s</strong>\n", title)
}

func (exp *Exporter) RenderText(text []ast.Inline) string {
	if exp.Context().Params["lang"] == "fr" {
		text = frundis.InsertNbsps(exp, text)
	}
	return html.EscapeString(exp.BaseContext().InlinesToText(text))
}

func (exp *Exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	w := exp.Context().GetW()
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

func (exp *Exporter) TableOfContentsInfos(flags map[string]bool) {
	// (dominitoc, etc.) (useless for html)
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
		c = "em"
	} else {
		c = *cmd
	}
	return frundis.Mtag{Begin: html.EscapeString(begin), End: html.EscapeString(end), Cmd: c}
}
