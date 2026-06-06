package expr

import (
	"fmt"
	"strconv"
	"unicode"
)

type tokKind int

const (
	tkEOF tokKind = iota
	tkNumber
	tkString
	tkIdent  // bareword (true / false / null / function names)
	// v4: tkVarPath removed (no $-namespace). Slot retained to preserve const ordering.
	tkVarPathReservedDoNotUse
	tkLParen
	tkRParen
	tkComma
	tkPlus
	tkMinus
	tkStar
	tkSlash
	tkPercent
	tkEq    // ==
	tkNe    // !=
	tkLt    // <
	tkLe    // <=
	tkGt    // >
	tkGe    // >=
	tkAnd   // &&
	tkOr    // ||
	tkNot   // !
	tkQuestion
	tkColon
)

type token struct {
	kind tokKind
	val  string  // for ident/varpath/string
	num  float64 // for number
	pos  int     // byte offset for error msgs
}

type lexer struct {
	src []rune
	pos int
}

func newLexer(src string) *lexer { return &lexer{src: []rune(src)} }

func (l *lexer) peekRune() (rune, bool) {
	if l.pos >= len(l.src) {
		return 0, false
	}
	return l.src[l.pos], true
}

func (l *lexer) advance() rune {
	r := l.src[l.pos]
	l.pos++
	return r
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.src) && unicode.IsSpace(l.src[l.pos]) {
		l.pos++
	}
}

func (l *lexer) tokens() ([]token, error) {
	var out []token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
		if t.kind == tkEOF {
			return out, nil
		}
	}
}

func (l *lexer) next() (token, error) {
	l.skipSpace()
	start := l.pos
	if l.pos >= len(l.src) {
		return token{kind: tkEOF, pos: start}, nil
	}
	r := l.advance()

	switch r {
	case '(':
		return token{kind: tkLParen, pos: start}, nil
	case ')':
		return token{kind: tkRParen, pos: start}, nil
	case ',':
		return token{kind: tkComma, pos: start}, nil
	case '+':
		return token{kind: tkPlus, pos: start}, nil
	case '-':
		return token{kind: tkMinus, pos: start}, nil
	case '*':
		return token{kind: tkStar, pos: start}, nil
	case '/':
		return token{kind: tkSlash, pos: start}, nil
	case '%':
		return token{kind: tkPercent, pos: start}, nil
	case '?':
		return token{kind: tkQuestion, pos: start}, nil
	case ':':
		return token{kind: tkColon, pos: start}, nil
	case '=':
		if l.matchRune('=') {
			return token{kind: tkEq, pos: start}, nil
		}
		return token{}, fmt.Errorf("expr: unexpected '=' at col %d (use '==' for equality)", start)
	case '!':
		if l.matchRune('=') {
			return token{kind: tkNe, pos: start}, nil
		}
		return token{kind: tkNot, pos: start}, nil
	case '<':
		if l.matchRune('=') {
			return token{kind: tkLe, pos: start}, nil
		}
		return token{kind: tkLt, pos: start}, nil
	case '>':
		if l.matchRune('=') {
			return token{kind: tkGe, pos: start}, nil
		}
		return token{kind: tkGt, pos: start}, nil
	case '&':
		if l.matchRune('&') {
			return token{kind: tkAnd, pos: start}, nil
		}
		return token{}, fmt.Errorf("expr: unexpected '&' at col %d (use '&&')", start)
	case '|':
		if l.matchRune('|') {
			return token{kind: tkOr, pos: start}, nil
		}
		return token{}, fmt.Errorf("expr: unexpected '|' at col %d (use '||')", start)
	case '"':
		return l.readString(start)
	case '$':
		// v4: $-namespace removed (no $vars / $params). Variables / params
		// are routed into Expr via bare-identifier inputs fed by GetVar / GetParam nodes.
		return token{}, fmt.Errorf("expr: '$' not allowed (v4) at col %d — use GetVar / GetParam node + Expr input pin", start)
	}

	if unicode.IsDigit(r) || (r == '.' && l.pos < len(l.src) && unicode.IsDigit(l.src[l.pos])) {
		l.pos-- // 退一步让 readNumber 起点对齐
		return l.readNumber(start)
	}

	if isIdentStart(r) {
		l.pos-- // 让 readIdent 起点对齐
		return l.readIdent(start)
	}

	return token{}, fmt.Errorf("expr: unexpected character %q at col %d", r, start)
}

func (l *lexer) matchRune(want rune) bool {
	if l.pos < len(l.src) && l.src[l.pos] == want {
		l.pos++
		return true
	}
	return false
}

func (l *lexer) readNumber(start int) (token, error) {
	for l.pos < len(l.src) && (unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '.') {
		l.pos++
	}
	s := string(l.src[start:l.pos])
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return token{}, fmt.Errorf("expr: bad number %q at col %d: %v", s, start, err)
	}
	return token{kind: tkNumber, num: n, pos: start}, nil
}

func (l *lexer) readString(start int) (token, error) {
	var b []rune
	for l.pos < len(l.src) {
		r := l.advance()
		if r == '"' {
			return token{kind: tkString, val: string(b), pos: start}, nil
		}
		if r == '\\' && l.pos < len(l.src) {
			esc := l.advance()
			switch esc {
			case 'n':
				b = append(b, '\n')
			case 't':
				b = append(b, '\t')
			case '\\':
				b = append(b, '\\')
			case '"':
				b = append(b, '"')
			default:
				b = append(b, esc)
			}
			continue
		}
		b = append(b, r)
	}
	return token{}, fmt.Errorf("expr: unterminated string starting at col %d", start)
}

// v4: readVarPath / tkVarPath removed. $-prefix is rejected at the dispatch site (single
// implementation path — no $vars/$sys/$params namespace exists in v4 grammar).
func (l *lexer) readIdent(start int) (token, error) {
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.pos++
	}
	return token{kind: tkIdent, val: string(l.src[start:l.pos]), pos: start}, nil
}

func isIdentStart(r rune) bool { return r == '_' || unicode.IsLetter(r) }
func isIdentPart(r rune) bool  { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }
