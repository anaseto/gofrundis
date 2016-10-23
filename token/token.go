package token

type Token int

const (
	ESCAPE   Token = iota // \X
	IESCAPE               // \*[...]
	AESCAPE               // \$<digit>
	NAESCAPE              // \$[escape]
	NFESCAPE              // \$?[flag]
	TEXT
	MACRO_NAME
	MACRO_END
	ARG_END
	COMMENT
	EXTEND_LINE
	ILLEGAL
	EOF
)

func (t Token) String() string {
	switch t {
	case ESCAPE:
		return "ESCAPE"
	case IESCAPE:
		return "IESCAPE"
	case AESCAPE:
		return "AESCAPE"
	case NAESCAPE:
		return "NAESCAPE"
	case NFESCAPE:
		return "NFESCAPE"
	case TEXT:
		return "TEXT"
	case MACRO_NAME:
		return "MACRO_NAME"
	case ARG_END:
		return "ARG_END"
	case ILLEGAL:
		return "ILLEGAL"
	case COMMENT:
		return "COMMENT"
	case MACRO_END:
		return "MACRO_END"
	case EXTEND_LINE:
		return "EXTEND_LINE"
	case EOF:
		return "EOF"
	}
	return ""
}
