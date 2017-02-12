package frundis

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/anaseto/gofrundis/ast"
)

// Error writes msgs to ctx.Werror with some additional context information.
func (ctx *Context) Error(msgs ...interface{}) {
	var s string
	if ctx.uMacroCall.loc != nil {
		file := ctx.uMacroCall.loc.curFile
		b := ctx.uMacroCall.loc.curBlocks[ctx.uMacroCall.loc.curBlock].(*ast.Macro)
		s = fmt.Sprint("frundis:", file, ":", b.Line,
			":in user macro `.", b.Name, "':")
	} else if ctx.loc != nil {
		if ctx.loc.curBlock >= 0 {
			b := ctx.block()
			line := b.GetLine()
			s = fmt.Sprint("frundis:", ctx.loc.curFile, ":", line, ":")
		} else {
			s = fmt.Sprint("frundis:", ctx.loc.curFile, ":")
		}
	} else {
		s = fmt.Sprint("frundis:")
	}
	s += ctx.Macro + ":"
	s += fmt.Sprint(msgs...)
	fmt.Fprintln(ctx.Werror, s)
}

// block returns current block.
func (ctx *Context) block() ast.Block {
	return ctx.loc.curBlocks[ctx.loc.curBlock]
}

// isValidFormat checks whether format is a valid format.
func (ctx *Context) isValidFormat(format string) bool {
	for _, f := range ctx.validFormats {
		if format == f {
			return true
		}
	}
	return false
}

// checkFormats warns if a format from formats is unknown.
func (ctx *Context) checkFormats(formats []string) {
	for _, f := range formats {
		if !ctx.isValidFormat(f) {
			ctx.Error("invalid argument to -f option:", f)
		}
	}
}

// notExportFormat tests whether none of the formats in formats is current
// export format.
func (ctx *Context) notExportFormat(formats []string) bool {
	for _, f := range formats {
		if f == ctx.Format {
			return false
		}
	}
	return true
}

// containsSpace checks wether a string contains any unicode space.
func containsSpace(s string) bool {
	for _, c := range s {
		if unicode.IsSpace(c) {
			return true
		}
	}
	return false
}

// isPunctArg tests wether arg contains only punctuation. If it starts with a
// \& escape, it is never considered punctuation.
func (ctx *Context) isPunctArg(arg []ast.Inline) bool {
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
	for _, c := range ctx.InlinesToText(arg) {
		if !unicode.IsPunct(c) {
			return false
		}
	}
	return true
}

// IsTrue returns true if string is empty or "0".
func IsTrue(s string) bool {
	return !(s == "" || s == "0")
}

// inlineToText converts an inline element to a string by processing escapes.
func (ctx *Context) inlineToText(elt ast.Inline) string {
	var res string
	switch elt := elt.(type) {
	case ast.Escape:
		res = elt.ToText()
	case ast.VarEscape:
		var ok bool
		res, ok = ctx.ivars[string(elt)]
		if !ok {
			ctx.Error("unknown variable name:", string(elt))
		}
	case ast.Text:
		res = string(elt)
	}
	return res
}

// InlinesToText stringifies a slice of inline elements, performing variable
// interpolation and simple escaping rules.
func (ctx *Context) InlinesToText(elts []ast.Inline) string {
	ctx.bufi2t.Reset()
	for _, elt := range elts {
		ctx.bufi2t.WriteString(ctx.inlineToText(elt))
	}
	return ctx.bufi2t.String()
}

// argsToText stringifies a list of arguments args using separator sep.
func argsToText(exp BaseExporter, args [][]ast.Inline, sep string) string {
	ctx := exp.Context()
	ctx.bufa2t.Reset()
	for i, arg := range args {
		if i > 0 {
			ctx.bufa2t.WriteString(sep)
		}
		ctx.bufa2t.WriteString(ctx.InlinesToText(arg))
	}
	return ctx.bufa2t.String()
}

// renderArgs renders a list of arguments.
func renderArgs(exp Exporter, args [][]ast.Inline) string {
	ctx := exp.Context()
	ctx.bufra.Reset()
	for i, arg := range args {
		if i > 0 {
			ctx.bufra.WriteRune(' ')
		}
		ctx.bufra.WriteString(exp.RenderText(arg))
	}
	return ctx.bufra.String()
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

// loXEntryInfos populates LoX structures with loXinfo information of a given
// class.
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

// getCommand returns a command from a list of arguments. If there is only one
// argument, and it contains spaces, this argument is passed to the shell
// as-is.
func getCommand(args []string) *exec.Cmd {
	var cmd *exec.Cmd
	if len(args) == 1 && containsSpace(args[0]) {
		cmd = exec.Command("/bin/sh", "-c", args[0])
	} else {
		cmd = exec.Command(args[0], args[1:]...)
	}
	return cmd
}

// shellFilter runs a filter on text using arguments args as the filtering
// command.
func shellFilter(exp Exporter, args []string, text string) string {
	ctx := exp.Context()
	if !ctx.Unrestricted {
		ctx.Error("skipping disallowed external command")
		return ""
	}
	file, err := ioutil.TempFile("", "frundis-")
	defer func() {
		file.Close()
		err := os.Remove(file.Name())
		if err != nil {
			ctx.Error("could not remove temporary file:", file.Name())
		}
	}()
	if err != nil {
		ctx.Error(err)
		return ""
	}
	_, err = file.WriteString(text)
	if err != nil {
		ctx.Error(err)
		return ""
	}
	file.Sync()
	file.Seek(0, 0) // return to start of the file
	cmd := getCommand(args)
	cmd.Stdin = file
	cmd.Stderr = os.Stderr
	bytes, err := cmd.Output()
	if err != nil {
		ctx.Error("shell command:", args, ":", err)
		return ""
	}
	return string(bytes)
}

// InsertNbsps inserts non-breaking spaces following french punctuation rules.
func InsertNbsps(exp Exporter, text []ast.Inline) []ast.Inline {
	ctx := exp.Context()
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
						ctx.Error("incorrect regular space before '", fmt.Sprintf("%c", c), "'")
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
							ctx.Error("incorrect regular space after '", fmt.Sprintf("%c", c), "'")
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

// readPairs reads a string s of pairs delimited by occurrences of the first
// character. It returns a list of strings of even length, or nil if s has not
// the correct format.
func readPairs(s string) ([]string, error) {
	sr := strings.NewReader(s)
	r, size, err := sr.ReadRune()
	if err != nil {
		return nil, err
	}
	s = s[size:]
	repls := strings.Split(s, fmt.Sprintf("%c", r))
	if len(repls)%2 != 0 {
		return nil, errors.New(fmt.Sprintf("odd number of items in '%s'", s))
	}
	return repls, nil
}

func testPairs(ctx *Context, pairs []string) {
	for i := 0; i < len(pairs)-1; i += 2 {
		if pairs[i] == "" {
			ctx.Error(fmt.Sprintf("in -a option:key %d is empty", (i/2)+1))
		}
		if strings.ContainsAny(pairs[i], "\"'>/=") {
			ctx.Error(fmt.Sprintf("in -a option:key %d contains invalid characters", (i/2)+1))
		}
		for _, c := range pairs[i] {
			if unicode.IsSpace(c) {
				ctx.Error(fmt.Sprintf("in -a option:key %d contains space", (i/2)+1))
			}
		}
	}
}
