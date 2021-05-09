package ast

// Block represents a macro line or a text block.
type Block interface {
	ImplementsBlock()
	GetLine() int
}

// Inline represents an inline element in a block, such as text or an escape.
type Inline interface {
	ImplementsInline()
}

// Macro represents data associated with a macro line.
type Macro struct {
	Name string
	Args [][]Inline
	Line int
}

// TextBlock represents data associated with a text block.
type TextBlock struct {
	Text []Inline
	Line int
}

func (m *Macro) ImplementsBlock()     {}
func (t *TextBlock) ImplementsBlock() {}

func (m *Macro) GetLine() int     { return m.Line }
func (t *TextBlock) GetLine() int { return t.Line }

type (
	// Escape represents a regular escape sequence
	Escape string
	// ArgEscape represents an argument escape (\$N)
	ArgEscape int
	// NamedArgEscape represents a named argument escape
	NamedArgEscape string
	// NamedArgEscape represents a flag escape
	NamedFlagEscape string
	// ArgEscape represents a variable interpolation escape
	VarEscape string
	// Text represents a bunch of inline text.
	Text string
)

func (e Escape) ImplementsInline()          {}
func (e ArgEscape) ImplementsInline()       {}
func (e NamedArgEscape) ImplementsInline()  {}
func (e NamedFlagEscape) ImplementsInline() {}
func (e VarEscape) ImplementsInline()       {}
func (t Text) ImplementsInline()            {}

func (e Escape) ToText() string {
	switch e {
	case "&":
		return ""
	case "e":
		return "\\"
	case "~":
		return string(rune(0xa0))
	default:
		return ""
	}
}
