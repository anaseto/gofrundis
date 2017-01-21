package mom

import (
	"fmt"
	"io/ioutil"

	"github.com/anaseto/gofrundis/frundis"
)

func (exp *exporter) beginMomDocument() {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	title := ctx.Params["document-title"]
	author := ctx.Params["document-author"]
	date := ctx.Params["document-date"]
	preamble := ctx.Params["mom-preamble"]
	var chapterString string
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
	fmt.Fprintf(ctx.W, ".PAPER A5\n")
	fmt.Fprintf(ctx.W, ".PRINTSTYLE TYPESET\n")
	fmt.Fprintf(ctx.W, ".TITLE \"%s\"\n", title)
	fmt.Fprintf(ctx.W, ".AUTHOR \"%s\"\n", author)
	fmt.Fprintf(ctx.W, ".\\\" %s\n", date)
	fmt.Fprintf(ctx.W, ".ATTRIBUTE_STRING \"\"\n")
	fmt.Fprintf(ctx.W, ".HEADING_STYLE 1 SPACE_AFTER\n")
	fmt.Fprintf(ctx.W, ".HEADING_STYLE 2 SPACE_AFTER\n")
	fmt.Fprintf(ctx.W, ".CHAPTER 1\n")
	switch ctx.Params["lang"] {
	case "fr":
		chapterString = "Chapitre"
	case "es":
		chapterString = "Capítulo"
	default:
		chapterString = "Chapter"
		// TODO: More? Or just let the user change this with custom
		// preamble.
	}
	fmt.Fprintf(ctx.W, ".CHAPTER_STRING \"%s\"\n", chapterString)
end:
	ctx.W.WriteString(".START\n")
}
