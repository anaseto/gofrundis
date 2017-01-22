package mom

import (
	"fmt"
	"io/ioutil"

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
				bctx.Error(err) // XXX use another function
			} else {
				ctx.W.Write(source)
				goto end
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
	fmt.Fprintf(ctx.W, ".PAPER A5\n")
	fmt.Fprintf(ctx.W, ".PRINTSTYLE TYPESET\n")
	fmt.Fprintf(ctx.W, ".TITLE \"%s\"\n", escape.Roff(title))
	fmt.Fprintf(ctx.W, ".AUTHOR \"%s\"\n", escape.Roff(author))
	fmt.Fprintf(ctx.W, ".\\\" %s\n", escape.Roff(date))
	fmt.Fprintf(ctx.W, ".ATTRIBUTE_STRING \"\"\n")
	fmt.Fprintf(ctx.W, ".HEADERS OFF\n")
	fmt.Fprintf(ctx.W, ".HEADING_STYLE 1 SIZE +6 QUAD C SPACE_AFTER NUMBER \n")
	fmt.Fprintf(ctx.W, ".HEADING_STYLE 2 SIZE +5 QUAD C SPACE_AFTER NUMBER \n")
	fmt.Fprintf(ctx.W, ".HEADING_STYLE 3 SIZE +3 SPACE_AFTER NUMBER \n")
	fmt.Fprintf(ctx.W, ".HEADING_STYLE 4 SIZE +2 SPACE_AFTER NUMBER \n")
end:
	ctx.W.WriteString(".START\n")
}
