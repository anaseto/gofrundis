package latex

import (
	"fmt"
	"io/ioutil"

	"github.com/anaseto/gofrundis/frundis"
)

func latexHeaderName(name string) string {
	var cmd string
	switch name {
	case "Pt":
		cmd = "part"
	case "Ch":
		cmd = "chapter"
	case "Sh":
		cmd = "section"
	case "Ss":
		cmd = "subsection"
	}
	return cmd
}

var miniLangs = map[string]string{
	"af": "afrikaans",
	"bg": "bulgarian",
	"br": "breton",
	"ca": "catalan",
	"cs": "czech",
	"cy": "welsh",
	"da": "danish",
	"de": "german",
	"el": "greek",
	"en": "english",
	"eo": "esperanto",
	"es": "spanish",
	"et": "estonian",
	"eu": "basque",
	"fi": "finnish",
	"fr": "french",
	"ga": "irish",
	"gd": "scottish",
	"gl": "galician",
	"he": "hebrew",
	"hr": "croatian",
	"hu": "magyar",
	"ia": "interlingua",
	"is": "icelandic",
	"it": "italian",
	"la": "latin",
	"nl": "dutch",
	"no": "norsk",
	"pl": "polish",
	"pt": "portuges",
	"ro": "romanian",
	"ru": "russian",
	"se": "samin",
	"sk": "slovak",
	"sl": "slovene",
	"sr": "serbian",
	"sv": "swedish",
	"tr": "turkish",
	"uk": "ukrainian"}

var babelLangs map[string]string

func Init() {
	babelLangs = make(map[string]string)
	for k, v := range miniLangs {
		babelLangs[k] = v
	}
	babelLangs["de"] = "ngerman"
	babelLangs["fr"] = "frenchb"
}

func (exp *exporter) beginLatexDocument() {
	ctx := exp.Context()
	bctx := exp.BaseContext()
	lang := ctx.Params["lang"]
	langBabel := babelLangs[lang]
	if langBabel == "" {
		langBabel = "english"
	}
	langMini := miniLangs[lang]
	if langMini == "" {
		langMini = "english"
	}
	title := ctx.Params["document-title"]
	author := ctx.Params["document-author"]
	date := ctx.Params["document-date"]
	preamble := ctx.Params["latex-preamble"]
	if preamble != "" {
		p, ok := frundis.SearchIncFile(exp, preamble)
		if !ok {
			bctx.Error("latex preamble:", preamble, ":no such file")
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
	if ctx.TocInfo.HasPart || ctx.TocInfo.HasChapter {
		ctx.W.WriteString("\\documentclass[a4paper,11pt]{book}\n")
	} else {
		ctx.W.WriteString("\\documentclass[a4paper,11pt]{article}\n")
	}
	if frundis.IsTrue(ctx.Params["latex-xelatex"]) {
		ctx.W.WriteString("\\usepackage{fontspec}\n")
		ctx.W.WriteString("\\usepackage{xunicode}\n")
		ctx.W.WriteString("\\usepackage{polyglossia}\n")
		fmt.Fprintf(ctx.W, "\\setmainlanguage{%s}\n", langBabel) // XXX do language names always be the same?
	} else {
		ctx.W.WriteString("\\usepackage[T1]{fontenc}\n")
		ctx.W.WriteString("\\usepackage[utf8]{inputenc}\n")
		fmt.Fprintf(ctx.W, "\\usepackage[%s]{babel}\n", langBabel)
	}
	if exp.minitoc {
		fmt.Fprintf(ctx.W, "\\usepackage[%s]{minitoc}\n", langMini)
	}
	if ctx.HasVerse {
		ctx.W.WriteString("\\usepackage{verse}\n")
	}
	if ctx.HasImage {
		ctx.W.WriteString("\\usepackage{graphicx}\n")
	}
	ctx.W.WriteString(`\usepackage{verbatim}
\usepackage[linkcolor=blue,colorlinks=true]{hyperref}
`)
	fmt.Fprintf(ctx.W, "\\title{%s}\n", title)
	fmt.Fprintf(ctx.W, "\\author{%s}\n", author)
	fmt.Fprintf(ctx.W, "\\date{%s}\n", date)
end:
	ctx.W.WriteString("\\begin{document}\n")
	if exp.dominilof {
		ctx.W.WriteString("\\dominilof\n")
	}
	if exp.dominilot {
		ctx.W.WriteString("\\dominilot\n")
	}
	if exp.dominitoc {
		ctx.W.WriteString("\\dominitoc\n")
	}
	if frundis.IsTrue(ctx.Params["title-page"]) {
		ctx.W.WriteString("\\maketitle\n")
	}
}

func (exp *exporter) EndLatexDocument() {
	ctx := exp.Context()
	ctx.W.WriteString("\n\\end{document}\n")
}
