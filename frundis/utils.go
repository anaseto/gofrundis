package frundis

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"unicode"
	"unicode/utf8"

	"github.com/anaseto/gofrundis/ast"
)

// Error writes msgs to bctx.Werror with some additional context information.
func (bctx *BaseContext) Error(msgs ...interface{}) {
	var s string
	if bctx.callInfo.loc != nil {
		file := bctx.callInfo.loc.curFile
		b := bctx.callInfo.loc.curBlocks[bctx.callInfo.loc.curBlock].(*ast.Macro)
		s = fmt.Sprint("frundis:", file, ":", b.Line,
			":in user macro `.", b.Name, "':")
	} else if bctx.loc != nil {
		if bctx.loc.curBlock >= 0 {
			b := bctx.block()
			line := b.GetLine()
			s = fmt.Sprint("frundis:", bctx.loc.curFile, ":", line, ":")
		} else {
			s = fmt.Sprint("frundis:", bctx.loc.curFile, ":")
		}
	} else {
		s = fmt.Sprint("frundis:")
	}
	s += bctx.macro + ":"
	s += fmt.Sprint(msgs...)
	fmt.Fprintln(bctx.Werror, s)
}

// block returns current block.
func (bctx *BaseContext) block() ast.Block {
	return bctx.loc.curBlocks[bctx.loc.curBlock]
}

// isValidFormat checks whether format is a valid format.
func (bctx *BaseContext) isValidFormat(format string) bool {
	for _, f := range bctx.validFormats {
		if format == f {
			return true
		}
	}
	return false
}

// checkFormats warns if a format from formats is unknown.
func (bctx *BaseContext) checkFormats(formats []string) {
	for _, f := range formats {
		if !bctx.isValidFormat(f) {
			bctx.Error("invalid argument to -f option:", f)
		}
	}
}

// notExportFormat tests whether none of the formats in formats is current
// export format.
func (bctx *BaseContext) notExportFormat(formats []string) bool {
	for _, f := range formats {
		if f == bctx.Format {
			return false
		}
	}
	return true
}

// isPunctArg tests wether arg contains only punctuation. If it starts with a
// \& escape, it is never considered punctuation.
func (bctx *BaseContext) isPunctArg(arg []ast.Inline) bool {
	if len(arg) > 0 {
		escape, ok := arg[0].(ast.Escape)
		if ok {
			switch string(escape) {
			case "&":
				return false
			case "~":
				arg = arg[1:]
			}
		}
	}
	if len(arg) == 0 {
		return false
	}
	for _, c := range bctx.InlinesToText(arg) {
		if !unicode.IsPunct(c) {
			return false
		}
	}
	return true
}

func (bctx *BaseContext) inlineToText(elt ast.Inline) string {
	var res string
	switch elt := elt.(type) {
	case ast.Escape:
		res = elt.ToText()
	case ast.VarEscape:
		var ok bool
		res, ok = bctx.vars[string(elt)]
		if !ok {
			bctx.Error("unknown variable name:", string(elt))
		}
	case ast.Text:
		res = string(elt)
	}
	return res
}

// InlinesToText stringifies a slice of inline elements, performing variable
// interpolation and simple escaping rules.
func (bctx *BaseContext) InlinesToText(elts []ast.Inline) string {
	bctx.bufi2t.Reset()
	for _, elt := range elts {
		bctx.bufi2t.WriteString(bctx.inlineToText(elt))
	}
	return bctx.bufi2t.String()
}

// argsToText stringifies a list of arguments args using separator sep.
func argsToText(exp BaseExporter, args [][]ast.Inline, sep string) string {
	bctx := exp.BaseContext()
	bctx.bufa2t.Reset()
	for i, arg := range args {
		if i > 0 {
			bctx.bufa2t.WriteString(sep)
		}
		bctx.bufa2t.WriteString(bctx.InlinesToText(arg))
	}
	return bctx.bufa2t.String()
}

// renderArgs renders a list of arguments.
func renderArgs(exp Exporter, args [][]ast.Inline) string {
	bctx := exp.BaseContext()
	bctx.bufra.Reset()
	for i, arg := range args {
		if i > 0 {
			bctx.bufra.WriteRune(' ')
		}
		bctx.bufra.WriteString(exp.RenderText(arg))
	}
	return bctx.bufra.String()
}

// getClosePunct returns a punctuation delimiter or an empty string, and an
// updated arguments slice.
func getClosePunct(exp Exporter, args [][]ast.Inline) ([][]ast.Inline, string) {
	ctx := exp.BaseContext()
	last := args[len(args)-1]
	hasPunct := false
	if ctx.isPunctArg(last) {
		hasPunct = true
		args = args[:len(args)-1]
	}
	var punct string
	if hasPunct {
		punct = exp.RenderText(last)
	}
	return args, punct
}

// SearchIncFile returns the path to filename relative to the current directory
// or the FRUNDISLIB environnment variable, and boolean true if such a file
// exists. Otherwise it returns a false boolean.
func SearchIncFile(exp Exporter, filename string) (string, bool) {
	ctx := exp.Context()
	if fi, err := os.Stat(filename); err == nil && fi.Mode().IsRegular() {
		return filename, true
	}
	for _, dir := range ctx.frundisINC {
		fpath := path.Join(dir, filename)
		fi, err := os.Stat(fpath)
		if err == nil && fi.Mode().IsRegular() {
			return fpath, true
		}
	}
	return filename, false
}

func containsSpace(s string) bool {
	for _, c := range s {
		if unicode.IsSpace(c) {
			return true
		}
	}
	return false
}

func loXEntryInfos(exp Exporter, class string, loXinfo *LoXinfo, id string) {
	ctx := exp.Context()
	loX, ok := ctx.LoXInfo[class]
	if !ok {
		loX = make(map[string]*LoXinfo)
		ctx.LoXInfo[class] = loX
	}
	loXinfo.Ref = exp.GenRef(loXinfo.RefPrefix, id, false)
	loX[loXinfo.Title] = loXinfo
	_, okStack := ctx.LoXstack[class]
	if okStack {
		ctx.LoXstack[class] = append(ctx.LoXstack[class], loXinfo)
	} else {
		ctx.LoXstack[class] = []*LoXinfo{loXinfo}
	}
}

func shellFilter(exp Exporter, shellcmd string, text string) string {
	bctx := exp.BaseContext()
	file, err := ioutil.TempFile("", "frundis-")
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()
	if err != nil {
		bctx.Error(err)
		return ""
	}
	_, err = file.WriteString(text)
	if err != nil {
		bctx.Error(err)
		return ""
	}
	file.Sync()
	file.Seek(0, 0) // return to start of the file
	cmd := exec.Command("/bin/sh", "-c", shellcmd)
	cmd.Stdin = file
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		bctx.Error(err)
		return ""
	}
	return string(bytes)
}

// InsertNbsps inserts non-breaking spaces following french punctuation rules.
func InsertNbsps(exp Exporter, text []ast.Inline) []ast.Inline {
	bctx := exp.BaseContext()
	newtext := []ast.Inline{}
	noinsertnbsp := false
	for i, elt := range text {
		switch elt := elt.(type) {
		case ast.Escape:
			noinsertnbsp = (elt == "&" || elt == "~")
			newtext = append(newtext, elt)
		case ast.Text:
			start := 0
			space := false
			for j, c := range elt {
				switch c {
				case '!', ':', ';', '?', 0xbb:
					if space {
						bctx.Error("incorrect regular space before '", fmt.Sprintf("%c", c), "'")
					}
					if !noinsertnbsp {
						if start != j {
							newtext = append(newtext, ast.Text(elt[start:j]), ast.Escape("~"))
						} else {
							newtext = append(newtext, ast.Escape("~"))
						}
						start = j
					}
					noinsertnbsp = false
				case 0xa0:
					noinsertnbsp = true
				case 0xab:
					next := j + utf8.RuneLen(0xab)
					if next <= len(elt)-1 {
						r, _ := utf8.DecodeRuneInString(string(elt[next:]))
						if r != 0xa0 {
							newtext = append(newtext, ast.Text(elt[start:next]), ast.Escape("~"))
							start = next
						}
						if r == ' ' {
							bctx.Error("incorrect regular space after '", fmt.Sprintf("%c", c), "'")
						}

					} else if i < len(text)-1 {
						switch text[i+1] {
						case ast.Escape("&"), ast.Escape("~"):
						default:
							newtext = append(newtext, ast.Text(elt[start:next]), ast.Escape("~"))
							start = next
						}
					}
				default:
					noinsertnbsp = false
				}
				space = c == ' ' || c == '\n'
			}
			if start <= len(elt)-1 {
				newtext = append(newtext, ast.Text(elt[start:len(elt)]))
			}
		default:
			newtext = append(newtext, elt)
		}
	}
	return newtext
}

// GetW returns a writer to be used in place of ctx.W in macro methods.
func (ctx *Context) GetW() io.Writer {
	if ctx.inpar {
		return &ctx.buf
	}
	return ctx.W
}

func parEnd(exp Exporter) {
	ctx := exp.Context()
	ctx.W.Write(exp.FormatParagraph(ctx.buf.Bytes()))
	ctx.buf.Reset()
	ctx.inpar = false
}

// IsTrue returns true if string is empty or "0".
func IsTrue(s string) bool {
	return !(s == "" || s == "0")
}
