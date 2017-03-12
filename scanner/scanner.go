package scanner

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/anaseto/gofrundis/token"
)

// Scanner gathers information and methods for scanning frundis files.
type Scanner struct {
	File    string    // file beeing read (for error messages only)
	Reader  io.Reader // reader to scan from
	Werror  io.Writer // where to output non-fatal errors
	bReader *bufio.Reader
	buf     bytes.Buffer // buffer
	ch      rune         // current character
	col     int          // current column number
	line    int          // current line number
	prevcol int          // previous line last column number
	state   scannerState // scanner state (e.g. expecting text block, argument, etc.)
}

type scannerState int

const (
	scanBlockStart scannerState = iota
	scanQuotedArg
	scanNewArg
	scanArgEnd
	scanArgMore
	scanCommentLine
	scanMacroName
	scanTextBlock
	scanEnd
)

func (s scannerState) String() string {
	switch s {
	case scanBlockStart:
		return "scanBlockStart"
	case scanQuotedArg:
		return "scanQuotedArg"
	case scanArgMore:
		return "scanArgMore"
	case scanNewArg:
		return "scanNewArg"
	case scanArgEnd:
		return "scanArgEnd"
	case scanMacroName:
		return "scanMacroName"
	case scanTextBlock:
		return "scanTextBlock"
	case scanEnd:
		return "scanEnd"
	}
	return ""
}

// Init initializes the Scanner.
func (s *Scanner) Init() {
	s.bReader = bufio.NewReader(s.Reader)
	s.line = 1
	if s.Werror == nil {
		s.Werror = os.Stderr
	}
}

func (s *Scanner) error(msg string) {
	line := s.line
	col := s.col
	if s.ch == '\n' {
		line--
		col = s.prevcol
	}
	fmt.Fprintf(s.Werror, "frundis:%s:%d:%d:%s\n", s.File, line, col, msg)
}

func (s *Scanner) scanMacroName() (token.Token, string) {
	// s.ch == '.'
	s.next()
	s.skipWhiteSpace()
	if s.ch == '\n' {
		s.state = scanNewArg
		return token.MACRO_NAME, ""
	}
	if s.ch == '\\' {
		s.next()
		if s.ch == '"' {
			s.state = scanCommentLine
			return token.MACRO_NAME, ""
		}
		s.skipLine()
		s.error("expecting '\"' of comment escape")
		return token.ILLEGAL, string(s.ch)
	}
	s.buf.Reset()
	for !unicode.IsSpace(s.ch) && s.ch >= 0 {
		if s.ch == '\\' {
			s.error("'\\' escape not allowed in macro name. Ignoring it.")
			s.next()
			continue
		}
		s.buf.WriteRune(s.ch)
		s.next()
	}
	return token.MACRO_NAME, s.buf.String()
}

func (s *Scanner) scanArgument() (token.Token, string) {
	switch s.ch {
	case '\\':
		if s.state == scanNewArg {
			s.state = scanArgMore
		}
		return s.scanEscape()
	case '\n':
		if s.state == scanArgMore {
			s.state = scanArgEnd
			return token.TEXT, ""
		} else if s.state == scanQuotedArg {
			s.error("unterminated quoted argument")
			s.state = scanArgEnd
			return token.TEXT, ""
		}
		s.next()
		if s.ch < 0 {
			return token.EOF, ""
		}
		s.state = scanBlockStart
		return token.MACRO_END, ""
	default:
		if s.state == scanNewArg {
			if s.ch == '"' {
				s.state = scanQuotedArg
				s.next()
			} else {
				s.state = scanArgMore
			}
		}
		return s.scanArgText()
	}
}

func (s *Scanner) scanArgText() (token.Token, string) {
	// s.ch is '"' or other character not among '\\' and '\n'
	s.buf.Reset()
scanAgain:
	if s.ch < 0 {
		s.state = scanEnd
		return token.TEXT, s.buf.String()
	}
	switch s.ch {
	case '\\':
		return token.TEXT, s.buf.String()
	case '"':
		if s.state == scanQuotedArg {
			s.next()
			if s.ch == '"' {
				s.buf.WriteRune('"')
				s.next()
				goto scanAgain
			} else {
				s.state = scanArgEnd
				return token.TEXT, s.buf.String()
			}
		} else {
			s.buf.WriteRune('"')
			s.next()
			goto scanAgain
		}
	default:
		if s.state != scanQuotedArg && unicode.IsSpace(s.ch) {
			s.state = scanArgEnd
			return token.TEXT, s.buf.String()
		} else if s.ch == '\n' {
			s.error("unterminated quoted argument")
			s.state = scanArgEnd
			return token.TEXT, s.buf.String()
		}
		s.buf.WriteRune(s.ch)
		s.next()
		goto scanAgain
	}
}

func (s *Scanner) scanTextBlock() (token.Token, string) {
	switch s.ch {
	case '\\':
		return s.scanEscape()
	default:
		return s.scanText()
	}
}

func (s *Scanner) scanText() (token.Token, string) {
	// s.ch != '\\'
	s.buf.Reset()
scanAgain:
	if s.ch < 0 {
		s.state = scanEnd
		return token.TEXT, s.buf.String()
	}
	switch s.ch {
	case '\\':
		return token.TEXT, s.buf.String()
	default:
		if s.ch == '\n' {
			s.next()
			if s.ch == '.' {
				s.state = scanMacroName
				return token.TEXT, s.buf.String()
			}
			if s.ch >= 0 {
				s.buf.WriteRune('\n')
			}
			goto scanAgain
		}
		s.buf.WriteRune(s.ch)
		s.next()
		goto scanAgain
	}
}

func (s *Scanner) scanEscape() (tok token.Token, lit string) {
	// s.ch == '\\'
	s.next()
	switch s.ch {
	case '\n':
		tok = token.EXTEND_LINE // XXX useful when in text block?
		s.next()
		if s.state == scanArgMore {
			s.skipWhiteSpace()
			s.state = scanNewArg
		}
	case '"':
		tok, lit = s.scanComment()
		// \n remains in when parsing a Textblock
	case 'e', '&', '~':
		tok = token.ESCAPE
		lit = string(s.ch)
		s.next()
	case '*':
		tok, lit = s.scanInterpolation(true, interpolation)
		if tok != token.ILLEGAL {
			s.next()
		}
	case '$':
		tok, lit = s.scanArgInterpolation()
		if tok != token.ILLEGAL {
			s.next()
		}
	default:
		tok = token.ESCAPE
		s.error(fmt.Sprintf("unknown escape:\\%c", s.ch))
		s.next()
	}
	return
}

type namedEscapeType int

const (
	interpolation namedEscapeType = iota
	namedArgument
	namedFlag
)

func (s *Scanner) scanInterpolation(next bool, t namedEscapeType) (token.Token, string) {
	// s.ch == '*' or '[' or '?'
	if next {
		// case '*', '?'
		s.next()
	}
	if s.ch != '[' {
		s.error("expecting '['")
		return token.ILLEGAL, ""
	}
	s.next()
	s.buf.Reset()
	for s.ch != ']' {
		if unicode.IsSpace(s.ch) {
			s.error("unexpected space character")
			return token.ILLEGAL, ""
		}
		if s.ch < 0 {
			s.error("unexpected EOF")
			return token.ILLEGAL, ""
		}
		s.buf.WriteRune(s.ch)
		s.next()
	}
	var tok token.Token
	switch t {
	case interpolation:
		tok = token.IESCAPE
	case namedArgument:
		tok = token.NAESCAPE
	case namedFlag:
		tok = token.NFESCAPE
	}
	return tok, s.buf.String()
}

func (s *Scanner) scanArgInterpolation() (token.Token, string) {
	// s.ch == '$'
	s.next()
	if s.ch <= '0' || '9' < s.ch {
		switch s.ch {
		case '@':
			return token.ESCAPE, "$@"
		case '[':
			return s.scanInterpolation(false, namedArgument)
		case '?':
			return s.scanInterpolation(true, namedFlag)
		}
		s.error("argument interpolation: expecting digit in range 1-9")
		return token.ILLEGAL, ""
	}
	// XXX allow more than one digit?
	return token.AESCAPE, string(s.ch)
}

func (s *Scanner) scanComment() (token.Token, string) {
	// s.ch == '"'
	s.next()
	s.buf.Reset()
	for s.ch != '\n' && s.ch >= 0 {
		s.buf.WriteRune(s.ch)
		s.next()
	}
	if s.state != scanTextBlock {
		if s.ch == '\n' {
			s.next()
		}
		s.state = scanBlockStart
	}
	return token.COMMENT, s.buf.String()
}

func (s *Scanner) skipWhiteSpace() {
	for unicode.IsSpace(s.ch) && s.ch != '\n' {
		s.next()
	}
}

func (s *Scanner) skipLine() {
	for s.ch != '\n' && s.ch >= 0 {
		s.next()
	}
}

// Scan returns next token, the line of source where it starts, and a string.
func (s *Scanner) Scan() (tok token.Token, line int, lit string, err error) {
	if s.ch == 0 {
		s.next()
	}
	line = s.line
scanAgain:
	switch s.state {
	case scanBlockStart:
		switch s.ch {
		case '.':
			s.state = scanMacroName
			goto scanAgain
		default:
			s.state = scanTextBlock
			goto scanAgain
		}
	case scanMacroName:
		tok, lit = s.scanMacroName()
		if s.state == scanCommentLine {
			break
		}
		s.skipWhiteSpace()
		if s.state != scanBlockStart && s.state != scanEnd {
			s.state = scanNewArg
		}
	case scanCommentLine:
		s.state = scanBlockStart
		tok, lit = s.scanComment()
	case scanArgMore, scanNewArg, scanQuotedArg:
		tok, lit = s.scanArgument()
	case scanArgEnd:
		s.skipWhiteSpace()
		s.state = scanNewArg
		tok, lit = token.ARG_END, ""
	case scanTextBlock:
		tok, lit = s.scanTextBlock()
	case scanEnd:
		tok, lit = token.EOF, ""
	default:
		// should not happen
		err = errors.New(fmt.Sprint("scanner.Scan:unhandled state:", s.state))
	}

	//fmt.Fprintf(os.Stderr, "%s,«%s»\n", tok, lit)
	return
}

func (s *Scanner) next() {
	r, _, err := s.bReader.ReadRune()
	if err != nil {
		s.ch = -1 // end of file
		s.state = scanEnd
		if err != io.EOF {
			s.error(err.Error())
		}
		return
	}
	//fmt.Printf("[%c]", r)
	s.ch = r
	if r == '\n' {
		s.line++
		s.prevcol = s.col
		s.col = 0
	} else {
		s.col++
	}
}
