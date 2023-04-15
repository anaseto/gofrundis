package latex

import (
	"os"
	"text/template"

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

func init() {
	babelLangs = make(map[string]string)
	for k, v := range miniLangs {
		babelLangs[k] = v
	}
	babelLangs["de"] = "ngerman"
}

func (exp *exporter) beginLatexDocument() {
	ctx := exp.Context()
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
	variant := ctx.Params["latex-variant"]
	switch variant {
	case "", "pdflatex", "xelatex":
	default:
		ctx.Errorf("unknown latex variant: %s", variant)
	}
	data := &struct {
		Title     string
		Author    string
		Date      string
		Book      bool
		XeLaTeX   bool
		MiniToc   bool
		HasVerse  bool
		HasImage  bool
		DominiLof bool
		DominiLot bool
		DominiToc bool
		LangBabel string
		LangMini  string
		TitlePage bool
	}{
		Title:     title,
		Author:    author,
		Date:      date,
		Book:      ctx.Toc.HasPart || ctx.Toc.HasChapter,
		XeLaTeX:   variant == "xelatex",
		MiniToc:   exp.minitoc,
		HasVerse:  ctx.Verse.Used,
		HasImage:  len(ctx.Images) > 0,
		DominiLof: exp.dominilof,
		DominiLot: exp.dominilot,
		DominiToc: exp.dominitoc,
		LangBabel: langBabel,
		LangMini:  langMini,
		TitlePage: frundis.IsTrue(ctx.Params["title-page"])}
	tmplBeginDocument, err := template.New("begin-document").Parse(`\begin{document}
{{if .DominiLof -}}
\dominilof
{{end -}}
{{if .DominiLot -}}
\dominilot
{{end -}}
{{if .DominiToc -}}
\dominitoc
{{end -}}
{{if .TitlePage -}}
\maketitle
{{end -}}
`)
	if err != nil {
		ctx.Error("internal error:", err)
		return
	}
	if preamble != "" {
		p, ok := frundis.SearchIncFile(exp, preamble)
		if !ok {
			ctx.Errorf("latex preamble: %s: no such file", preamble)
		} else {
			source, err := os.ReadFile(p)
			if err != nil {
				ctx.Error(err)
			} else {
				ctx.Wout.Write(source)
				err = tmplBeginDocument.Execute(ctx.Wout, data)
				if err != nil {
					ctx.Error("internal error:", err)
				}
				return
			}
		}
	}
	tmpl, err := template.New("preamble").Parse(`
{{- if .Book -}}
\documentclass[a4paper,11pt]{book}
{{else -}}
\documentclass[a4paper,11pt]{article}
{{end -}}
{{if .XeLaTeX -}}
\usepackage{fontspec}
\usepackage{xunicode}
\usepackage{polyglossia}
\setmainlanguage{ {{- .LangBabel -}} }
{{else -}}
\usepackage[T1]{fontenc}
\usepackage[utf8]{inputenc}
\usepackage[{{.LangBabel}}]{babel}
{{end -}}
{{if .MiniToc -}}
\usepackage[{{.LangMini}}]{minitoc}
{{end -}}
{{if .HasVerse -}}
\usepackage{verse}
{{end -}}
{{if .HasImage -}}
\usepackage{graphicx}
{{end -}}
\usepackage{verbatim}
\usepackage[linkcolor=blue,colorlinks=true]{hyperref}
\title{ {{- .Title -}} }
\author{ {{- .Author -}} }
\date{ {{- .Date -}} }
`)
	if err != nil {
		ctx.Error("internal error:", err)
		return
	}
	err = tmpl.Execute(ctx.Wout, data)
	if err != nil {
		ctx.Error(err)
	}
	err = tmplBeginDocument.Execute(ctx.Wout, data)
	if err != nil {
		ctx.Error("internal error:", err)
	}
}

func (exp *exporter) EndLatexDocument() {
	ctx := exp.Context()
	ctx.Wout.WriteString("\n\\end{document}\n")
}
