package mom

import (
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/anaseto/gofrundis/frundis"
)

func (exp *exporter) beginMomDocument() {
	ctx := exp.Context()
	title := ctx.Params["document-title"]
	author := ctx.Params["document-author"]
	date := ctx.Params["document-date"]
	preamble := ctx.Params["mom-preamble"]
	if preamble != "" {
		p, ok := frundis.SearchIncFile(exp, preamble)
		if !ok {
			ctx.Errorf("mom preamble: %s: no such file", preamble)
		} else {
			source, err := ioutil.ReadFile(p)
			if err != nil {
				ctx.Error(err)
			} else {
				ctx.Wout.Write(source)
				ctx.Wout.WriteString(".START\n")
				return
			}
		}
	}
	switch ctx.Params["lang"] {
	case "fr":
		fmt.Fprintf(ctx.Wout, ".do hla fr\n")
		fmt.Fprintf(ctx.Wout, ".do hpf hyphen.fr\n")
	default:
		fmt.Fprintf(ctx.Wout, ".do hla us\n")
		fmt.Fprintf(ctx.Wout, ".do hpf hyphen.us\n")
		fmt.Fprintf(ctx.Wout, ".do hpfa hyphenex.us\n")
	}
	data := &struct {
		Title  string
		Author string
		Date   string
	}{
		title,
		author,
		date}
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
		ctx.Error("internal error:", err)
		return
	}
	err = tmpl.Execute(ctx.Wout, data)
	if err != nil {
		ctx.Error(err)
	}
	ctx.Wout.WriteString(".START\n")
}
