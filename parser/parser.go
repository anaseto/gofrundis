package parser

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/scanner"
	"github.com/anaseto/gofrundis/token"
)

type Parser struct {
	Source string    // for error messages location information (e.g. filename)
	Werror io.Writer // where non-fatal scanning error messages go (default os.Stderr)
	line   int
	lit    string
	scan   *scanner.Scanner
	tok    token.Token
}

// ParseWithReader parses a frundis source from a reader and returns a list of
// AST blocks.
func (p *Parser) ParseWithReader(reader io.Reader) ([]ast.Block, error) {
	s := &scanner.Scanner{Reader: reader, File: p.Source, Werror: p.Werror}
	s.Init()
	p.scan = s
	p.initialize()
	blocks := []ast.Block{}
	for {
		b, err := p.parseBlock()
		if err != nil {
			return blocks, err
		}
		if b == nil {
			break
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// ParseFile parses a frundis file and returns a list of AST blocks. Sets
// p.Source to filename if empty.
func (p *Parser) ParseFile(filename string) ([]ast.Block, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(file)
	if p.Source == "" {
		p.Source = filename
	}
	return p.ParseWithReader(reader)
}

// ParseString parses a frundis string and returns a list of AST blocks
func (p *Parser) ParseString(str string) ([]ast.Block, error) {
	reader := strings.NewReader(str)
	return p.ParseWithReader(reader)
}

// Initialize newly created parser data (scanner current character and first
// current token)
func (p *Parser) initialize() error {
	p.line = 1
	var err error
	p.tok, p.line, p.lit, err = p.scan.Scan() // initialize parser
	return err
}

// Returns next block from parser
func (p *Parser) parseBlock() (ast.Block, error) {
	var b ast.Block
	var err error
	switch p.tok {
	case token.MACRO_NAME:
		b, err = p.parseMacro()
	case token.TEXT, token.ESCAPE, token.IESCAPE, token.AESCAPE, token.NAESCAPE, token.NFESCAPE, token.COMMENT, token.ILLEGAL:
		b, err = p.parseText()
	case token.EOF:
		b = nil
	default:
		err = fmt.Errorf("parser.parseBlock:unexpected token:%#v\n", p.tok)
	}
	//fmt.Fprintf(os.Stderr, "%#v\n", b)
	return b, err
}

func (p *Parser) parseText() (*ast.TextBlock, error) {
	// p.tok != token.MACRO_NAME && p.tok != token.EOF
	b := []ast.Inline{}
	line := p.line
parse:
	for {
		switch p.tok {
		case token.TEXT:
			if p.lit != "" {
				b = append(b, ast.Text(p.lit))
			}
		case token.ESCAPE:
			b = append(b, ast.Escape(p.lit))
		case token.IESCAPE:
			b = append(b, ast.VarEscape(p.lit))
		case token.AESCAPE:
			n, _ := strconv.ParseInt(p.lit, 10, 0) // we know it is a digit
			b = append(b, ast.ArgEscape(n))
		case token.NAESCAPE:
			b = append(b, ast.NamedArgEscape(p.lit))
		case token.NFESCAPE:
			b = append(b, ast.NamedFlagEscape(p.lit))
		case token.COMMENT, token.ILLEGAL:
		case token.MACRO_NAME, token.EOF:
			break parse
		default:
			return nil, fmt.Errorf("parser.parseText:unexpected token:%#v", p.tok)
		}
		var err error
		p.tok, p.line, p.lit, err = p.scan.Scan()
		if err != nil {
			return nil, err
		}
	}
	return &ast.TextBlock{Text: b, Line: line}, nil
}

func (p *Parser) parseMacro() (*ast.Macro, error) {
	// p.tok == token.MACRO_NAME
	m := ast.Macro{Name: p.lit, Line: p.line}
	a := []ast.Inline{}
parse:
	for {
		var err error
		p.tok, p.line, p.lit, err = p.scan.Scan()
		if err != nil {
			return nil, err
		}
		switch p.tok {
		case token.TEXT:
			if p.lit != "" {
				a = append(a, ast.Text(p.lit))
			}
		case token.ESCAPE:
			a = append(a, ast.Escape(p.lit))
		case token.IESCAPE:
			a = append(a, ast.VarEscape(p.lit))
		case token.AESCAPE:
			n, _ := strconv.ParseInt(p.lit, 10, 0) // we know it is a digit
			a = append(a, ast.ArgEscape(n))
		case token.NAESCAPE:
			a = append(a, ast.NamedArgEscape(p.lit))
		case token.NFESCAPE:
			a = append(a, ast.NamedFlagEscape(p.lit))
		case token.COMMENT, token.MACRO_END, token.EOF:
			if len(a) > 0 {
				m.Args = append(m.Args, a)
			}
			if p.tok != token.EOF {
				p.tok, p.line, p.lit, err = p.scan.Scan()
				if err != nil {
					return nil, err
				}
			}
			break parse
		case token.EXTEND_LINE, token.ILLEGAL:
		case token.ARG_END:
			m.Args = append(m.Args, a)
			a = []ast.Inline{}
		default:
			return nil, fmt.Errorf("parser.parseMacro:unexpected token:%#v", p.tok)
		}
	}
	return &m, nil
}
