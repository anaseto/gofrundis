package xhtml

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/frundis"
)

type toc int

const (
	xhtmlToc toc = iota
	ncxToc
	navToc
)

func (exp *exporter) writeTOC(w io.Writer, toctype toc, opts map[string][]ast.Inline, flags map[string]bool) {
	ctx := exp.Context()
	tocStack := ctx.LoXstack["toc"]
	if len(tocStack) == 0 {
		ctx.Error("warning:no TOC information found, skipping TOC generation")
		return
	}
	flags["toc"] = true
	start := 0
	miniMacro := "Ch"
	tocInfo := ctx.Toc
	if flags["mini"] && tocInfo.NavCount() > 0 {
		navEntry := ctx.LoXstack["nav"][tocInfo.NavCount()-1]
		start = navEntry.Count
		miniMacro = navEntry.Macro
	}
	var closeList, closeItem string
	switch toctype {
	case xhtmlToc:
		closeList = "</ul>"
		closeItem = "</li>"
	case ncxToc:
		closeList = ""
		closeItem = "</navPoint>"
	case navToc:
		closeList = "</ol>"
		closeItem = "</li>"
	}

	// TOC top
	switch toctype {
	case xhtmlToc:
		fmt.Fprint(w, "<div class=\"toc\">\n")
		var title string
		if t, ok := opts["title"]; flags["mini"] || ok {
			title = exp.RenderText(t)
		} else {
			title = ctx.Params["document-title"]
		}
		if title != "" {
			fmt.Fprintf(w, "  <h2 id=\"toc-title\" class=\"toc-title\">%s</h2>\n", title)
		}
		fmt.Fprint(w, "  <ul>\n")
	case ncxToc:
		fmt.Fprint(w, "<navMap>\n")
		title := ctx.Params["document-title"]
		fmt.Fprint(w, "    <navPoint id=\"titlepage\">\n")
		fmt.Fprintf(w, "      <navLabel><text>%s</text></navLabel>\n", title)
		fmt.Fprint(w, "      <content src=\"index.xhtml\" />\n")
		fmt.Fprint(w, "    </navPoint>\n")
	case navToc:
		fmt.Fprint(w, "<nav epub:type=\"toc\" id=\"navtoc\">\n")
		fmt.Fprint(w, "  <ol>\n")
		title := ctx.Params["document-title"]
		if title != "" {
			fmt.Fprintf(w, "    <li><a href=\"index.xhtml\" class=\"toc-title\">%s</a></li>\n", title)
		}
	}

	// TOC entries
	// level: the actual depth level of the entry in TOC.
	// titleLevel: the level of the title (1 for Pt, 2 for Ch, etc.)
	// previousTitleLevel: the level of the previous title
	level := 0 // 0 for first iteration
	previousTitleLevel := 1
	for i := start; i < len(tocStack); i++ {
		entry := tocStack[i]
		macro := entry.Macro
		if flags["mini"] {
			if macro == miniMacro || macro == "Pt" {
				break
			}
		}
		if flags["summary"] {
			if flags["mini"] && miniMacro == "Ch" {
				if macro != "Sh" {
					continue
				}
			} else {
				if macro != "Pt" && macro != "Ch" {
					continue
				}
			}
		}
		titleLevel := ctx.Toc.HeaderLevel(macro)

		// Computation of level and previousTitleLevel
		switch {
		case level == 0:
			level = 1
			previousTitleLevel = titleLevel
		case titleLevel > previousTitleLevel:
			diference := titleLevel - previousTitleLevel
			switch toctype {
			case xhtmlToc:
				fmt.Fprint(w, strings.Repeat("  ", level+1), "<ul>\n")
			case navToc:
				fmt.Fprint(w, strings.Repeat("  ", level+1), "<ol>\n")
			}
			previousTitleLevel = titleLevel
			level = level + diference
		case titleLevel < previousTitleLevel:
			diference := titleLevel - previousTitleLevel
			if diference+level < 1 {
				diference = 1 - level
			}
			fmt.Fprintf(w, "%s%s\n", strings.Repeat("  ", level+1), closeItem)
			for j := level; j > level+diference; j-- {
				fmt.Fprintf(w, "%s%s%s\n", strings.Repeat("  ", j), closeList, closeItem)
			}
			previousTitleLevel = titleLevel
			level = level + diference
		case titleLevel == previousTitleLevel:
			fmt.Fprintf(w, "%s%s\n", strings.Repeat("  ", level+1), closeItem)
		}

		// Print entry
		switch toctype {
		case xhtmlToc:
			exp.xhtmlTOClikeEntry(w, entry, flags, level)
		case ncxToc:
			num := entry.Num
			href := entry.Ref
			if num != "" {
				num += ". "
			}
			id := strings.Replace(href, "#", "-", -1)
			fmt.Fprintf(w, "%s<navPoint id=\"%s\">\n", strings.Repeat("  ", level+1), id)
			fmt.Fprint(w, strings.Repeat("  ", level+2),
				"<navLabel><text>", num, entry.Title, "</text></navLabel>\n")
			fmt.Fprintf(w, "%s<content src=\"%s\" />\n", strings.Repeat("  ", level+2), href)
		case navToc:
			num := entry.Num
			href := entry.Ref
			if num != "" {
				num += ". "
			}
			fmt.Fprintf(w, "%s<li><a href=\"%s\">%s%s</a>\n",
				strings.Repeat("  ", level+1), href, num, entry.Title)
		}
	}
	if level > 0 {
		fmt.Fprintf(w, "%s%s\n", strings.Repeat("  ", level+1), closeItem)
	}
	for i := level; i > 1; i-- {
		fmt.Fprintf(w, "%s%s%s\n", strings.Repeat("  ", i), closeList, closeItem)
	}

	// TOC bottom
	switch toctype {
	case xhtmlToc:
		fmt.Fprint(w, "  </ul>\n")
		fmt.Fprint(w, "</div>\n")
	case ncxToc:
		fmt.Fprint(w, "</navMap>\n")
	case navToc:
		fmt.Fprint(w, "  </ol>\n")
		fmt.Fprint(w, "</nav>\n")
	}
}

func (exp *exporter) xhtmlLoX(w io.Writer, class string) {
	ctx := exp.Context()
	switch class {
	case "lot", "lof", "lop":
	default:
		ctx.Errorf("warning:unknown List-of-X class:%s", class)
		return
	}
	tocStack := ctx.LoXstack[class]
	if len(tocStack) == 0 {
		ctx.Errorf("warning:no '%s' information found, skipping '%s' generation", class, class)
		return
	}
	fmt.Fprintf(w, "<div class=\"%s\">\n", class)
	fmt.Fprintf(w, "  <ul>\n")
	for _, entry := range tocStack {
		exp.xhtmlTOClikeEntry(w, entry, map[string]bool{}, 1)
	}
	fmt.Fprintf(w, "  </ul>\n")
	fmt.Fprintf(w, "</div>\n")
}

func (exp *exporter) xhtmlTOClikeEntry(w io.Writer, entry *frundis.LoXinfo, flags map[string]bool, level int) {
	href := entry.Ref
	var num string
	if !(flags["nonum"] || strings.HasPrefix(href, "index") && !exp.AllInOneFile) {
		if flags["toc"] {
			num = entry.Num
			if num != "" {
				num += ". "
			}
		} else {
			num = fmt.Sprintf("%d. ", entry.Count)
		}
	}
	fmt.Fprintf(w, "%s<li><a href=\"%s\">%s%s</a>\n",
		strings.Repeat("  ", level+1), href, num, entry.Title)
}

func (exp *exporter) getID(entry *frundis.LoXinfo) string {
	var id string
	if exp.AllInOneFile {
		id = fmt.Sprintf("s%d", entry.Count)
	} else {
		id = entry.Ref
		if strings.ContainsRune(id, '#') {
			id = strings.TrimLeftFunc(id, func(r rune) bool { return r != '#' })
			id = strings.TrimPrefix(id, "#")
		}
		if strings.ContainsRune(id, '.') {
			id = strings.TrimRightFunc(id, func(r rune) bool { return r != '.' })
			id = strings.TrimSuffix(id, ".")
		}
	}
	return id
}

func (exp *exporter) XHTMLandEPUBcommonHeader(w io.Writer) {
	ctx := exp.Context()
	epub3 := strings.HasPrefix(ctx.Params["epub-version"], "3")
	var xmlnsepub string
	if ctx.Format == "epub" && epub3 {
		fmt.Fprint(w, "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
		xmlnsepub = "xmlns:epub=\"http://www.idpf.org/2007/ops\" "
		xmlnsepub += fmt.Sprintf("xml:lang=\"%s\" ", ctx.Params["lang"])
	}
	if ctx.Format == "epub" && epub3 ||
		ctx.Format == "xhtml" && frundis.IsTrue(ctx.Params["xhtml5"]) {
		fmt.Fprint(w, "<!DOCTYPE html>\n")
	} else if ctx.Format == "xhtml" || !epub3 {
		fmt.Fprint(w, "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.1//EN\" \"http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd\">\n")
	}
	fmt.Fprintf(w, "<html xmlns=\"http://www.w3.org/1999/xhtml\" %slang=\"%s\">\n", xmlnsepub, ctx.Params["lang"])
	fmt.Fprint(w, "  <head>\n")
	if ctx.Format == "epub" && epub3 {
		fmt.Fprint(w, "    <meta charset=\"utf-8\" />\n")
	} else {
		fmt.Fprint(w, "    <meta http-equiv=\"Content-type\" content=\"text/html; charset=utf-8\" />\n")
	}
}

func (exp *exporter) XHTMLdocumentHeader(w io.Writer, title string) {
	ctx := exp.Context()
	exp.XHTMLandEPUBcommonHeader(w)
	if title != "" {
		fmt.Fprintf(w, "    <title>%s</title>\n", title)
	}
	if favicon, ok := ctx.Params["xhtml-favicon"]; ok && ctx.Format == "xhtml" {
		fmt.Fprintf(w, "    <link rel=\"shortcut icon\" type=\"image/x-icon\" href=\"%s\" />\n", favicon)
	}
	switch ctx.Format {
	case "epub":
		if _, ok := ctx.Params["epub-css"]; ok {
			fmt.Fprint(w, "    <link rel=\"stylesheet\" href=\"stylesheet.css\" />\n")
		}
	case "xhtml":
		if xhtmlcss, ok := ctx.Params["xhtml-css"]; ok {
			fmt.Fprintf(w, "    <link rel=\"stylesheet\" href=\"%s\" />\n", xhtmlcss)
		}
	}
	fmt.Fprint(w, `  </head>
  <body>
`)
	if xhtmltop, ok := ctx.Params["xhtml-top"]; ok && ctx.Format == "xhtml" {
		f, ok := frundis.SearchIncFile(exp, xhtmltop)
		if !ok {
			ctx.Errorf("xhtml-top: %s: no such file", xhtmltop)
		} else {
			data, err := ioutil.ReadFile(f)
			if err != nil {
				ctx.Errorf("xhtml-top: %s: %s", f, err)
				return
			}
			w.Write(data)
		}
	}
}

func (exp *exporter) XHTMLdocumentFooter(w io.Writer) {
	ctx := exp.Context()
	if xhtmlbottom, ok := ctx.Params["xhtml-bottom"]; ok && ctx.Format == "xhtml" {
		f, ok := frundis.SearchIncFile(exp, xhtmlbottom)
		if !ok {
			ctx.Errorf("xhtml-bottom: %s: no such file", xhtmlbottom)
		} else {
			data, err := ioutil.ReadFile(f)
			if err != nil {
				ctx.Errorf("xhtml-bottom: %s: %s", f, err)
				return
			}
			w.Write(data)
		}
	}
	w.Write([]byte(`  </body>
</html>
`))
}

var indexTranslations = map[string]string{
	"de": "Index",
	"en": "Index",
	"eo": "Indekso",
	"es": "Índice",
	"fr": "Index"}

func (exp *exporter) xhtmlFileOutputChange(title string) {
	ctx := exp.Context()
	if ctx.Format == "xhtml" && exp.xhtmlNavigationText.Len() > 0 {
		ctx.Wout.Write(exp.xhtmlNavigationText.Bytes())
		exp.xhtmlNavigationText.Reset()
	}
	exp.XHTMLdocumentFooter(ctx.Wout)
	ctx.Wout.Flush()
	fprefix, ok := ctx.Params["xhtml-chap-prefix"]
	if !ok {
		fprefix = "body"
	}
	idText := ""
	useID, ok := ctx.Params["xhtml-chap-prefix-ids"]
	if ok && (useID != "" && useID != "0") {
		idText = ctx.ID
	}
	if idText != "" {
		idText = "-" + idText
	}
	var outFile string
	switch ctx.Format {
	case "epub":
		outFile = path.Join(exp.OutputFile, "EPUB",
			fmt.Sprintf("%s-%d-%d%s.xhtml", fprefix, ctx.Toc.PartCount, ctx.Toc.ChapterCount, idText))
	case "xhtml":
		outFile = path.Join(exp.OutputFile,
			fmt.Sprintf("%s-%d-%d%s.html", fprefix, ctx.Toc.PartCount, ctx.Toc.ChapterCount, idText))
	}
	if exp.curOutputFile != nil {
		err := exp.curOutputFile.Close()
		if err != nil {
			ctx.Error("closing file:", err)
		}
	}
	var err error
	exp.curOutputFile, err = os.Create(outFile)
	if err != nil {
		ctx.Error("create file:", err)
	}
	if exp.curOutputFile != nil {
		ctx.Wout = bufio.NewWriter(exp.curOutputFile)
	} else {
		ctx.Wout = bufio.NewWriter(ioutil.Discard)
		exp.curOutputFile = os.Stdout // XXX
	}
	exp.XHTMLdocumentHeader(ctx.Wout, title)

	if ctx.Format == "epub" {
		return
	}
	// IF NOT EPUB
	toc := ctx.Toc
	navLoX := ctx.LoXstack["nav"]
	var previous *frundis.LoXinfo
	if !(toc.NavCount() <= 1) {
		previous = navLoX[toc.NavCount()-2]
	}
	var next *frundis.LoXinfo
	if !(toc.NavCount() >= len(navLoX)) {
		next = navLoX[toc.NavCount()]
	}

	exp.xhtmlNavigationText.Reset()
	exp.xhtmlNavigationText.WriteString(`    <div class="topnav">
      <ul class="topnav">
`)
	if previous != nil {
		href := previous.Ref
		fmt.Fprintf(exp.xhtmlNavigationText, "        <li><a href=\"%s\">&lt;</a></li>\n", href)
	} else {
		fmt.Fprint(exp.xhtmlNavigationText, "        <li>&lt;</li>\n")
	}
	index := html.EscapeString(ctx.Params["xhtml-go-up"])
	if index == "" {
		var ok bool
		index, ok = indexTranslations[ctx.Params["lang"]]
		if !ok {
			index = "\u2191"
		}
	}
	fmt.Fprintf(exp.xhtmlNavigationText, "        <li><a href=\"index.html\">%s</a></li>\n", index)
	if next != nil {
		href := next.Ref
		fmt.Fprintf(exp.xhtmlNavigationText, "        <li><a href=\"%s\">&gt;</a></li>\n", href)
	} else {
		fmt.Fprint(exp.xhtmlNavigationText, "        <li>&gt;</li>\n")
	}
	exp.xhtmlNavigationText.WriteString(`      </ul>
    </div>
`)
	ctx.Wout.Write(exp.xhtmlNavigationText.Bytes())
}

func (exp *exporter) xhtmlTitlePage() {
	ctx := exp.Context()
	if !frundis.IsTrue(ctx.Params["title-page"]) {
		return
	}
	if title := ctx.Params["document-title"]; title != "" {
		fmt.Fprintf(ctx.Wout, "<h1 class=\"title\">%s</h1>\n", title)
	} else {
		ctx.Error("warning:parameter ``title-page'' set to true value but no document title specified")
	}
	if author := ctx.Params["document-author"]; author != "" {
		fmt.Fprintf(ctx.Wout, "<h2 class=\"author\">%s</h2>\n", author)
	} else {
		ctx.Error("warning:parameter ``title-page'' set to true value but no document author specified")
	}
	if date := ctx.Params["document-date"]; date != "" {
		fmt.Fprintf(ctx.Wout, "<h3 class=\"date\">%s</h3>\n", date)
	} else {
		ctx.Error("warning:parameter ``title-page'' set to true value but no document date specified")
	}
}

func escapeFilter(text string) string {
	return html.EscapeString(text)
}
