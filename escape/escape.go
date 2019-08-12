package escape

import (
	"strings"
)

var latexEscapes = []string{
	"{", "\\{",
	"}", "\\}",
	"[", "[", // XXX do something?
	"]", "]",
	"%", "\\%",
	"&", "\\&",
	"$", "\\$",
	"#", "\\#",
	"_", "\\_",
	"^", "\\^{}",
	"\\", "\\textbackslash{}",
	"~", "\\~{}",
	string('\xa0'), "~"}

var latexEscaper = strings.NewReplacer(latexEscapes...)

// LaTeX escapes special LaTeX characters.
func LaTeX(text string) string {
	return latexEscaper.Replace(text)
}

// LaTeXPercent escapes just % (for post-processing url, for example).
func LaTeXPercent(text string) string {
	return strings.Replace(text, "%", "\\%", -1)
}

var markdownEscapes = []string{
	"*", "\\*",
	"`", "\\`",
	"_", "\\_",
	"#", "\\#",
	"[", "\\[",
	">", "\\>",
	"]", "\\]",
	"~", "\\~",
	"\\", "\\\\"}

var markdownEscaper = strings.NewReplacer(markdownEscapes...)

// Markdown escapes special markdown characters (incomplete).
func Markdown(text string) string {
	return markdownEscaper.Replace(text)
}

var roffEscapes = []string{
	"\"", "\\(dq",
	"…", "...", // bug with groff
	"'", "\\(cq",
	".", "\\&.",
	"\\", "\\e",
	string('\xa0'), "\\~",
}

var roffEscaper = strings.NewReplacer(roffEscapes...)

// Roff escapes special roff characters.
func Roff(text string) string {
	return roffEscaper.Replace(text)
}
