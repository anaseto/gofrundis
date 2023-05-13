package mom

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"codeberg.org/anaseto/gofrundis/ast"
	"codeberg.org/anaseto/gofrundis/escape"
	"codeberg.org/anaseto/gofrundis/frundis"
)

// Options gathers configuration for groff mom exporter.
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
	Ctx           *frundis.Context
	OutputFile    string
	curOutputFile *os.File
	Standalone    bool
	verse         bool
	inCell        bool
	fontstack     []string
}

func (exp *exporter) Init() {
	ctx := &frundis.Context{Wout: bufio.NewWriter(os.Stdout), Format: "mom"}
	exp.Ctx = ctx
	ctx.Init()
	ctx.Filters["escape"] = escape.Roff
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
	if exp.Standalone {
		exp.beginMomDocument()
	}
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
	w := exp.Context().W()
	if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
	}
	fmt.Fprint(w, ".LIST USER \"\"\n")
}

func (exp *exporter) BeginDescValue() {
	// do nothing
}

func (exp *exporter) BeginDialogue() {
	ctx := exp.Context()
	w := ctx.W()
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
	w := ctx.W()
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

func (exp *exporter) BeginEnumList(id string) {
	w := exp.Context().W()
	if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
	}
	fmt.Fprint(w, ".LIST\n")
}

func (exp *exporter) BeginHeader(macro string, numbered bool, title string) {
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
	w := ctx.W()
	if level < 3 {
		fmt.Fprintf(w, ".NEWPAGE\n")
	}
	fmt.Fprintf(w, ".HEADING %d NAMED s:%d \"", level, ctx.Toc.HeaderCount)
}

func (exp *exporter) BeginItem() {
	w := exp.Context().W()
	fmt.Fprint(w, ".ITEM\n")
}

func (exp *exporter) BeginEnumItem() {
	w := exp.Context().W()
	fmt.Fprint(w, ".ITEM\n")
}

func (exp *exporter) BeginItemList(id string) {
	w := exp.Context().W()
	if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
	}
	fmt.Fprint(w, ".LIST\n")
}

func (exp *exporter) BeginMarkupBlock(tag string, id string) {
	ctx := exp.Context()
	w := ctx.W()
	if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
	}
	mtag, okMtag := ctx.Mtags[tag]
	exp.fontstack = append(exp.fontstack, mtag.Cmd)
	if !okMtag {
		fmt.Fprint(w, "\\f[I]")
	} else {
		fmt.Fprintf(w, "\\f[%s]", mtag.Cmd)
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

func (exp *exporter) BeginTable(tableinfo *frundis.TableData) {
	w := exp.Context().W()
	lll := strings.Repeat("l ", tableinfo.Cols)
	if tableinfo.Title != "" {
		fmt.Fprintf(w, ".FLOAT\n")
	}
	fmt.Fprintf(w, ".TS\nallbox;\n%s.\n", lll)
}

func (exp *exporter) BeginTableCell() {
	ctx := exp.Context()
	if ctx.Table.Cell > 1 {
		fmt.Fprint(ctx.W(), "\t")
	}
	exp.inCell = true
}

func (exp *exporter) BeginTableRow() {
}

func (exp *exporter) BeginVerse(title string, id string) {
	w := exp.Context().W()
	exp.verse = true
	if title != "" {
		fmt.Fprintf(w, ".HEADING 5 \"%s\"\n", title)
		fmt.Fprintf(w, ".PDF_TARGET \"poem:%s\"\n", id)
	} else if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
	}
	fmt.Fprint(w, ".QUOTE_SIZE -1\n")
	fmt.Fprint(w, ".QUOTE_INDENT 1\n")
	fmt.Fprint(w, ".QUOTE\n")
}

func (exp *exporter) BeginVerseLine() {
}

func (exp *exporter) CheckParamAssignement(param string, value string) bool {
	return true
	// XXX: nothing to be done for now
}

func (exp *exporter) Context() *frundis.Context {
	return exp.Ctx
}

func (exp *exporter) CrossReference(idf frundis.IDInfo, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	switch idf.Type {
	case frundis.NoID:
		fmt.Fprintf(w, "%s%s", idf.Name, punct)
	default:
		fmt.Fprintf(w, ".PDF_LINK \"%s\" SUFFIX \"%s\" \"%s\"", idf.Ref, punct, idf.Name)
		// FIXME?: name could mess with surrounding markup if it has
		// markup (the problem is groff \f[..] that cannot simply be
		// reliably closed).
	}
}

func (exp *exporter) DescName(name string) {
	w := exp.Context().W()
	fmt.Fprintf(w, ".ITEM\n\\f[B]%s\\f[R]\n", name)
}

func (exp *exporter) EndDescList() {
	exp.Context().Wout.WriteString(".LIST OFF\n.PP\n")
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
			fmt.Fprintf(w, ".%s OFF\n", cmd)
		}
	}
}

func (exp *exporter) EndEnumList() {
	exp.Context().Wout.WriteString(".LIST OFF\n.PP\n")
}

func (exp *exporter) EndEnumItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndHeader(macro string, numbered bool, title string) {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, "\"\n")
	fmt.Fprint(w, ".PP\n")
	// if !numbered {
	// 	fmt.Fprintf(w, "\\addcontentsline{toc}{%s}{%s}\n", cmd, titleText)
	// }
}

func (exp *exporter) EndItemList() {
	exp.Context().Wout.WriteString(".LIST OFF\n.PP\n")
}

func (exp *exporter) EndItem() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndMarkupBlock(tag string, id string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	mtag, okMtag := ctx.Mtags[tag]
	if okMtag {
		fmt.Fprint(w, mtag.End)
	}
	exp.fontstack = exp.fontstack[:len(exp.fontstack)-1]
	cmd := "R"
	if len(exp.fontstack) > 0 {
		cmd = exp.fontstack[len(exp.fontstack)-1]
	}
	if ctx.Macro == "Em" && (ctx.PrevMacro == "Lk" || ctx.PrevMacro == "Sx") {
		// NOTE: this is quite hacky
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprintf(w, "\\f[%s]%s", cmd, punct)
}

func (exp *exporter) EndParagraph(pbreak frundis.ParagraphBreak) {
	w := exp.Context().W()
	switch pbreak {
	case frundis.ParBreakForced:
		if exp.verse {
			fmt.Fprint(w, "\n\n")
		}
	case frundis.ParBreakItem:
	case frundis.ParBreakBlock:
		fmt.Fprint(w, "\n.PP\n")
	default:

		if exp.verse {
			fmt.Fprint(w, "\n\n")
		} else {
			fmt.Fprint(w, "\n.PP\n")
		}
	}
}

func (exp *exporter) EndStanza() {
	exp.EndParagraph(frundis.ParBreakNormal)
}

func (exp *exporter) EndTable(tableinfo *frundis.TableData) {
	ctx := exp.Context()
	w := ctx.W()
	fmt.Fprint(w, ".TE\n")
	if tableinfo.Title != "" {
		fmt.Fprintf(w, ".CAPTION \"%s\" TO_LIST TABLES\n", tableinfo.Title)
		fmt.Fprintf(w, ".PDF_TARGET \"tbl:%d\"\n", ctx.Table.TitCount)
		fmt.Fprintf(w, ".FLOAT OFF\n")
	} else if tableinfo.ID != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", tableinfo.ID)
	}
}

func (exp *exporter) EndTableCell() {
	exp.inCell = false
}

func (exp *exporter) EndTableRow() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) EndVerse() {
	w := exp.Context().W()
	exp.verse = false
	fmt.Fprint(w, ".QUOTE OFF\n")
}

func (exp *exporter) EndVerseLine() {
	w := exp.Context().W()
	fmt.Fprint(w, "\n")
}

func (exp *exporter) FormatParagraph(text []byte) []byte {
	if exp.inCell {
		return bytes.Replace(text, []byte("\n"), []byte(" "), -1)
	}
	return text // TODO: do something?
}

func (exp *exporter) FigureImage(image string, caption string, link string, alt string) {
	ctx := exp.Context()
	w := ctx.W()
	_, err := os.Stat(image)
	if err != nil {
		ctx.Error("image not found:", image)
		return
	}
	ext := path.Ext(image)
	if ext != ".eps" && ext != ".pdf" {
		ctx.Error("expected .eps or .pdf but got ", image)
	}
	image = escape.Roff(image)
	fmt.Fprintf(w, ".FLOAT\n")
	fmt.Fprintf(w, ".PDF_IMAGE \"%s\"\n", image)
	fmt.Fprintf(w, ".CAPTION \"%s\" TO_LIST FIGURES\n", caption)
	fmt.Fprintf(w, ".PDF_TARGET \"fig:%d\"\n", ctx.FigCount)
	fmt.Fprintf(w, ".FLOAT OFF\n")
}

func (exp *exporter) GenRef(prefix string, id string, hasfile bool) string {
	if prefix != "" {
		return fmt.Sprintf("%s:%s", prefix, id)
	}
	return id
}

func (exp *exporter) HeaderReference(macro string) string {
	return exp.GenRef("s", strconv.Itoa(exp.Context().Toc.HeaderCount), false)
}

func (exp *exporter) InlineImage(image string, link string, id string, punct string, alt string) {
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
	ext := path.Ext(image)
	if ext != ".eps" && ext != ".pdf" {
		ctx.Error("expected .eps or .pdf but got ", image)
	}
	image = escape.Roff(image)
	fmt.Fprintf(w, ".PDF_IMAGE \"%s\"", image) // TODO: use punct
	if id != "" {
		fmt.Fprintf(w, ".PDF_TARGET \"%s\"\n", id)
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
		u = escape.Roff(parsedURL.String())
	}
	fmt.Fprintf(w, ".PDF_WWW_LINK %s SUFFIX \"%s\" \"%s\"", u, punct, escape.Roff(label))
	// XXX warn if label ends with '*' or '+' ? (they have special meaning
	// in mom at the end of the hotlink text)
}

func (exp *exporter) LkWithoutLabel(uri string, punct string) {
	ctx := exp.Context()
	w := ctx.W()
	parsedURL, err := url.Parse(uri)
	var u string
	if err != nil {
		ctx.Error("invalid url or path:", uri)
	} else {
		u = escape.Roff(parsedURL.String())
	}
	fmt.Fprintf(w, ".PDF_WWW_LINK %s SUFFIX \"%s\"", u, punct)
	// XXX: -ns option should be invalid after...
}

func (exp *exporter) ParagraphTitle(title string) {
	w := exp.Context().W()
	fmt.Fprintf(w, ".HEADING 5 PARAHEAD \"%s\"\n", title)
}

func (exp *exporter) RenderText(text []ast.Inline) string {
	ctx := exp.Context()
	switch ctx.Params["lang"] {
	case "fr":
		text = frundis.FrenchTypography(exp, text)
	case "en":
		text = frundis.EnglishTypography(exp, text)
	}
	return escape.Roff(ctx.InlinesToText(text))
}

func (exp *exporter) TableOfContents(opts map[string][]ast.Inline, flags map[string]bool) {
	// NOTE: mom table of contents does not play nicely with the frundis
	// way of doing table of contents, so leave this work to the user (it
	// is just a matter of adding a .TOC at the end of the document with a
	// format specific block).
}

func (exp *exporter) TableOfContentsInfos(flags map[string]bool) {
}

func (exp *exporter) Xdtag(cmd string, pairs []string) frundis.Dtag {
	return frundis.Dtag{Cmd: cmd}
}

func (exp *exporter) Xmtag(cmd *string, begin string, end string, pairs []string) frundis.Mtag {
	var c string
	if cmd == nil || *cmd == "" {
		c = "I"
	} else {
		c = *cmd
	}
	return frundis.Mtag{Begin: begin, End: end, Cmd: c}
}
