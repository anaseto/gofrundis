package markdown

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

var pbuf bytes.Buffer    // paragraph buffer
var wordbuf bytes.Buffer // word buffer

func processText(indent int, text []byte) []byte {
	pbuf.Reset()
	wordbuf.Reset()
	indentspaces := strings.Repeat(" ", indent)
	spaces := false
	col := 0
	wantSpace := false
	for _, c := range string(text) {
		if unicode.IsSpace(c) && c != 0xa0 {
			if !spaces && wordbuf.Len() > 0 {
				spaces = true
			}
			continue
		}
		if spaces && wordbuf.Len() > 0 {
			spaces = false
			w := wordbuf.Bytes()
			wlen := utf8.RuneCount(w)
			if col+wlen > 55 {
				if wantSpace {
					pbuf.WriteRune('\n')
					pbuf.WriteString(indentspaces)
					col = 0
				}
			} else {
				if wantSpace {
					pbuf.WriteRune(' ')
					col++
				}
			}
			pbuf.Write(w)
			wordbuf.Reset()
			col += wlen
			wantSpace = true
		}
		wordbuf.WriteRune(c)
	}
	if wordbuf.Len() > 0 {
		w := wordbuf.Bytes()
		wlen := utf8.RuneCount(w)
		if wantSpace {
			if wlen+col > 55 {
				pbuf.WriteRune('\n')
				pbuf.WriteString(indentspaces)
			} else {
				pbuf.WriteRune(' ')
			}
		}
		pbuf.Write(w)
	}
	return pbuf.Bytes()
}
