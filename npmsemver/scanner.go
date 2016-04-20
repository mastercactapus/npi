package npmsemver

import (
	"bufio"
	"bytes"
	"io"
)

type token int

var eof = rune(0)

const (
	TokenIllegal token = iota
	TokenEOF
	TokenWs

	TokenNumber
	TokenIdentifier
	TokenPlaceholder // x or X in semver

	TokenSeparator  // period, between identifiers/numbers
	TokenHyphen     // hyphen for version ranges
	TokenBuild      // plus as part of a version -- signifies metadata
	TokenPrerelease // hyphen part of a version -- signifies prerelease tag

	TokenTilde
	TokenCaret
	TokenGT
	TokenLT
	TokenEq
	TokenNot
	TokenOr
	TokenAsterisk
)

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}
func isNumber(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
func isIdent(ch rune) bool {
	return ch == '-' || isNumber(ch) || isLetter(ch)
}

type scanner struct {
	r       *bufio.Reader
	lastTok token
	vers    int
}

func newScanner(r io.Reader) *scanner {
	return &scanner{r: bufio.NewReader(r)}
}
func (s *scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}
func (s *scanner) unread() { _ = s.r.UnreadRune() }

func (s *scanner) scanWhitespace() (tok token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return TokenWs, buf.String()
}

func (s *scanner) scanIdent() (tok token, lit string) {
	var buf bytes.Buffer
	first:=s.read()
	buf.WriteRune(first)

	tok = TokenIdentifier

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isIdent(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	if s.vers < 3 && buf.Len() == 1 && (first == 'x' || first=='X') {
		return TokenPlaceholder, buf.String()
	}

	return tok, buf.String()
}

func (s *scanner) scanNumber() (tok token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	tok = TokenNumber

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isNumber(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return tok, buf.String()
}

func (s *scanner) Scan() (tok token, lit string) {
	ch := s.read()

	if isWhitespace(ch) {
		s.vers = 0
		s.unread()
		s.lastTok = TokenWs
		return s.scanWhitespace()
	} else if s.vers == 3 && ch == '-' {
		s.vers++
		s.lastTok = TokenPrerelease
		return TokenPrerelease, "-"
	} else if (s.vers == 3 || s.vers == 4) && ch == '+' {
		s.vers++
		s.lastTok = TokenBuild
		return TokenBuild, "+"
	} else if ch == '-' {
		s.vers = 0
		s.lastTok = TokenHyphen
		return TokenHyphen, "-"
	} else if (s.vers == 0 || s.lastTok == TokenSeparator) && s.vers < 3 && isNumber(ch) {
		s.unread()
		s.vers++
		s.lastTok = TokenNumber
		return s.scanNumber()
	} else if s.vers >= 3 && isIdent(ch) {
		s.unread()
		tok, lit = s.scanIdent()
		s.lastTok = tok
		return tok, lit
	} else if s.vers < 3 && ch == 'x' || ch == 'X' {
		s.vers++
		s.lastTok = TokenPlaceholder
		return TokenPlaceholder, string(ch)
	}

	switch ch {
	case eof:
		s.vers = 0
		return TokenEOF, ""
	case '>':
		s.vers = 0
		s.lastTok = TokenGT
		return TokenGT, ">"
	case '<':
		s.vers = 0
		s.lastTok = TokenLT
		return TokenLT, "<"
	case '|':
		s.vers = 0
		ch = s.read()
		if ch != '|' {
			s.lastTok = TokenIllegal
			return TokenIllegal, "|" + string(ch)
		}
		return TokenOr, "||"
	case '=':
		s.vers = 0
		s.lastTok = TokenEq
		return TokenEq, "="
	case '!':
		s.vers = 0
		s.lastTok = TokenNot
		return TokenNot, "!"
	case '~':
		s.vers = 0
		s.lastTok = TokenTilde
		return TokenTilde, "~"
	case '^':
		s.vers = 0
		s.lastTok = TokenCaret
		return TokenCaret, "^"
	case '.':
		if s.lastTok != TokenNumber && s.lastTok != TokenIdentifier {
			return TokenIllegal, string(ch)
		}
		s.lastTok = TokenSeparator
		return TokenSeparator, "."
	}

	return TokenIllegal, string(ch)

}
