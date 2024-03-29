package frundis

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"codeberg.org/anaseto/gofrundis/ast"
)

// Error writes msgs to ctx.Werror with some additional context information.
func (ctx *Context) Error(msgs ...interface{}) {
	if ctx.quiet {
		return
	}
	var s string
	if ctx.uMacroCall.loc != nil {
		file := ctx.uMacroCall.loc.curFile
		b := ctx.uMacroCall.loc.curBlocks[ctx.uMacroCall.loc.curBlock].(*ast.Macro)
		s = fmt.Sprint("frundis: ", file, ":", b.Line,
			":in user macro `.", b.Name, "':")
	} else if ctx.loc != nil {
		if ctx.loc.curBlock >= 0 && len(ctx.loc.curBlocks) > 0 {
			b := ctx.block()
			line := b.GetLine()
			s = fmt.Sprint("frundis: ", ctx.loc.curFile, ":", line, ":")
		} else {
			s = fmt.Sprint("frundis: ", ctx.loc.curFile, ":")
		}
	} else {
		s = "frundis: "
	}
	if ctx.Macro != "" {
		s += ctx.Macro + ": "
	}
	s += fmt.Sprintln(msgs...)
	fmt.Fprint(ctx.Werror, s)
}

// Errorf writes formatted msgs to ctx.Werror with some additional context information.
func (ctx *Context) Errorf(format string, msgs ...interface{}) {
	ctx.Error(fmt.Sprintf(format, msgs...))
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

// IsTrue returns true unless string is empty or "0".
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
			if len(string(elt)) > 0 && string(elt)[0] == '$' {
				res = os.Getenv(string(elt)[1:])
			} else {
				ctx.Error("unknown variable name:", string(elt))
			}
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
func argsToText(exp Exporter, args [][]ast.Inline) string {
	ctx := exp.Context()
	ctx.bufa2t.Reset()
	for i, arg := range args {
		if i > 0 {
			ctx.bufa2t.WriteRune(' ')
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
	loXinfo.Ref = exp.GenRef(loXinfo.RefPrefix, id, false)
	_, okStack := ctx.LoXstack[class]
	if okStack {
		ctx.LoXstack[class] = append(ctx.LoXstack[class], loXinfo)
	} else {
		ctx.LoXstack[class] = []*LoXinfo{loXinfo}
	}
	if loXinfo.ID != "" {
		var idtype IDType
		switch class {
		case "lof":
			idtype = FigureID
		case "lot":
			idtype = TableID
		case "lop":
			idtype = PoemID
		}
		ctx.storeID(loXinfo.ID, IDInfo{Ref: loXinfo.Ref, Name: loXinfo.Title, Type: idtype})
	}
}

// storeId stores an id with reference string ref, and of type idtype.
func (ctx *Context) storeID(id string, idinfo IDInfo) {
	if _, ok := ctx.IDs[id]; ok {
		q := ctx.quiet
		ctx.quiet = false
		ctx.Error("already used id")
		ctx.quiet = q
	}
	ctx.IDs[id] = idinfo
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
	file, err := os.CreateTemp("", "frundis-")
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
	err = file.Sync()
	if err != nil {
		ctx.Errorf("shell command: %v: file sync: %v", args, err)
		return ""
	}
	_, err = file.Seek(0, 0) // return to start of the file
	if err != nil {
		ctx.Errorf("shell command: %v: file seek: %v", args, err)
		return ""
	}
	cmd := getCommand(args)
	cmd.Stdin = file
	cmd.Stderr = os.Stderr
	bytes, err := cmd.Output()
	if err != nil {
		ctx.Errorf("shell command: %v: %v", args, err)
		return ""
	}
	return string(bytes)
}

// FrenchTypography inserts non-breaking spaces following french punctuation
// rules, as well as replacing apostrophes with typographic ones.
func FrenchTypography(exp Exporter, text []ast.Inline) []ast.Inline {
	ctx := exp.Context()
	newtext := []ast.Inline{}
	escape := false
	for i, elt := range text {
		switch elt := elt.(type) {
		case ast.Escape:
			escape = (elt == "&" || elt == "~")
			newtext = append(newtext, elt)
		case ast.Text:
			start := 0
			space := false
			for j, c := range elt {
				switch c {
				case '!', ':', ';', '?', 0xbb:
					if space {
						ctx.Errorf("incorrect regular space before '%c'", c)
					}
					if !escape {
						if start != j {
							newtext = append(newtext, ast.Text(elt[start:j]), ast.Escape("~"))
						} else {
							newtext = append(newtext, ast.Escape("~"))
						}
						start = j
					}
					escape = false
				case 0xa0:
					// XXX somewhat incorrect with respect to ’, but it shouldn't happen in practice.
					escape = true
				case 0xab:
					next := j + utf8.RuneLen(0xab)
					if next <= len(elt)-1 {
						r, _ := utf8.DecodeRuneInString(string(elt[next:]))
						if r != 0xa0 {
							newtext = append(newtext, ast.Text(elt[start:next]), ast.Escape("~"))
							start = next
						}
						if r == ' ' {
							ctx.Errorf("incorrect regular space after '%c'", c)
						}

					} else if i < len(text)-1 {
						switch text[i+1] {
						case ast.Escape("&"), ast.Escape("~"):
						default:
							newtext = append(newtext, ast.Text(elt[start:next]), ast.Escape("~"))
							start = next
						}
					} else {
						newtext = append(newtext, ast.Text(elt[start:next]), ast.Escape("~"))
						start = next
					}
				case '\'':
					next := j + utf8.RuneLen('\'')
					if !escape {
						if start != j {
							newtext = append(newtext, ast.Text(elt[start:j]))
						}
						newtext = append(newtext, ast.Text("’"))
						start = next
					}
					escape = false
				default:
					escape = false
				}
				space = c == ' ' || c == '\n'
			}
			if start <= len(elt)-1 {
				newtext = append(newtext, ast.Text(elt[start:]))
			}
		default:
			newtext = append(newtext, elt)
		}
	}
	return newtext
}

// EnglishTypography replaces apostrophes with typographic ones.
func EnglishTypography(exp Exporter, text []ast.Inline) []ast.Inline {
	newtext := []ast.Inline{}
	escape := false
	for _, elt := range text {
		switch elt := elt.(type) {
		case ast.Escape:
			escape = (elt == "&" || elt == "~")
			newtext = append(newtext, elt)
		case ast.Text:
			start := 0
			for j, c := range elt {
				switch c {
				case '\'':
					next := j + utf8.RuneLen('\'')
					if !escape {
						if start != j {
							newtext = append(newtext, ast.Text(elt[start:j]))
						}
						newtext = append(newtext, ast.Text("’"))
						start = next
					}
					escape = false
				default:
					escape = false
				}
			}
			if start <= len(elt)-1 {
				newtext = append(newtext, ast.Text(elt[start:]))
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
		return nil, err // this probably cannot happen
	}
	s = s[size:]
	repls := strings.Split(s, string(r))
	if len(repls)%2 != 0 {
		return nil, fmt.Errorf("odd number of items in '%s'", s)
	}
	return repls, nil
}

func checkPairs(ctx *Context, pairs []string) {
	for i := 0; i < len(pairs)-1; i += 2 {
		if pairs[i] == "" {
			ctx.Errorf("in -a option: key %d is empty", (i/2)+1)
		}
		if strings.ContainsAny(pairs[i], "\"'>/=") {
			ctx.Errorf("in -a option: key %d contains invalid characters", (i/2)+1)
		}
		for _, c := range pairs[i] {
			if unicode.IsSpace(c) {
				ctx.Errorf("in -a option: key %d contains space", (i/2)+1)
			}
		}
	}
}
