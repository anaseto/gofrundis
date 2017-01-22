package mom

import (
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/anaseto/gofrundis/escape"
	"github.com/anaseto/gofrundis/frundis"
)

func (exp *exporter) beginMomDocument() {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	title := ctx.Params["document-title"]
	author := ctx.Params["document-author"]
	date := ctx.Params["document-date"]
	preamble := ctx.Params["mom-preamble"]
	if preamble != "" {
		p, ok := frundis.SearchIncFile(exp, preamble)
		if !ok {
			bctx.Error("mom preamble:", preamble, ":no such file")
		} else {
			source, err := ioutil.ReadFile(p)
			if err != nil {
				bctx.Error(err)
			} else {
				ctx.W.Write(source)
				ctx.W.WriteString(".START\n")
				return
			}
		}
	}
	switch ctx.Params["lang"] {
	case "fr":
		fmt.Fprintf(ctx.W, ".do hla fr\n")
		fmt.Fprintf(ctx.W, ".do hpf hyphen.fr\n")
	default:
		fmt.Fprintf(ctx.W, ".do hla us\n")
		fmt.Fprintf(ctx.W, ".do hpf hyphen.us\n")
		fmt.Fprintf(ctx.W, ".do hpfa hyphenex.us\n")
	}
	data := &struct {
		Title  string
		Author string
		Date   string
	}{
		escape.Roff(title),
		escape.Roff(author),
		escape.Roff(date)}
	tmpl, err := template.New("preamble").Parse(`.PAPER A5
.PRINTSTYLE TYPESET
.TITLE "{{.Title}}"
.AUTHOR "{{.Author}}"
.\" {{.Date}}
.ATTRIBUTE_STRING ""
.HEADERS OFF
.HEADING_STYLE 1 SIZE +6 QUAD C SPACE_AFTER NUMBER
.HEADING_STYLE 2 SIZE +5 QUAD C SPACE_AFTER NUMBER
.HEADING_STYLE 3 SIZE +3 SPACE_AFTER NUMBER
.HEADING_STYLE 4 SIZE +2 SPACE_AFTER NUMBER
`)
	if err != nil {
		bctx.Error("internal error:", err)
		return
	}
	err = tmpl.Execute(ctx.W, data)
	if err != nil {
		bctx.Error(err)
	}
	ctx.W.WriteString(".START\n")
}
