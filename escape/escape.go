package escape

import (
	"fmt"
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
	fmt.Sprintf("%c", '\xa0'), "~"}

var latexEscaper = strings.NewReplacer(latexEscapes...)

// EscapeLatexString escapes special LaTeX characters.
func EscapeLatexString(text string) string {
	return latexEscaper.Replace(text)
}

// EscapeLatexPercent escapes just % (for post-processing url, for example).
func EscapeLatexPercent(text string) string {
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

// EscapeMarkdownString escapes special markdown characters (incomplete).
func EscapeMarkdownString(text string) string {
	return markdownEscaper.Replace(text)
}
