package ast

type Block interface {
	ImplementsBlock()
	GetLine() int
}

type Inline interface {
	ImplementsInline()
}

type Macro struct {
	Name string
	Args [][]Inline
	Line int
}

type TextBlock struct {
	Text []Inline
	Line int
}

func (m *Macro) ImplementsBlock()     {}
func (t *TextBlock) ImplementsBlock() {}

func (m *Macro) GetLine() int     { return m.Line }
func (t *TextBlock) GetLine() int { return t.Line }

type (
	Escape          string
	ArgEscape       int
	NamedArgEscape  string
	NamedFlagEscape string
	VarEscape       string
	Text            string
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
		return string(0xa0)
	default:
		return ""
	}
}
