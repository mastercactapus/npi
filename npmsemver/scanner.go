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

func (s *scanner) scanIdentNumber() (tok token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	tok = TokenNumber

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isIdent(ch) {
			s.unread()
			break
		} else if !isNumber(ch) {
			tok = TokenIdentifier
			buf.WriteRune(ch)
		} else {
			buf.WriteRune(ch)
		}
	}

	return tok, buf.String()
}

func (s *scanner) Scan() (tok token, lit string) {
	ch := s.read()

	if isWhitespace(ch) {
		s.unread()
		s.lastTok = TokenWs
		return s.scanWhitespace()
	} else if (s.lastTok == TokenIdentifier || s.lastTok == TokenNumber) && ch == '-' {
		s.lastTok = TokenPrerelease
		return TokenPrerelease, "-"
	} else if (s.lastTok == TokenIdentifier || s.lastTok == TokenNumber) && ch == '+' {
		s.lastTok = TokenBuild
		return TokenBuild, "+"
	} else if ch == '-' {
		s.lastTok = TokenHyphen
		return TokenHyphen, "-"
	} else if isIdent(ch) {
		s.unread()
		tok, lit = s.scanIdentNumber()
		s.lastTok = tok
		return tok, lit
	}

	switch ch {
	case eof:
		return TokenEOF, ""
	case '>':
		s.lastTok = TokenGT
		return TokenGT, ">"
	case '<':
		s.lastTok = TokenLT
		return TokenLT, "<"
	case '|':
		ch = s.read()
		if ch != '|' {
			s.lastTok = TokenIllegal
			return TokenIllegal, "|" + string(ch)
		}
		return TokenOr, "||"
	case '=':
		s.lastTok = TokenEq
		return TokenEq, "="
	case '!':
		s.lastTok = TokenNot
		return TokenNot, "!"
	case '~':
		s.lastTok = TokenTilde
		return TokenTilde, "~"
	case '^':
		s.lastTok = TokenCaret
		return TokenCaret, "^"
	case '.':
		s.lastTok = TokenSeparator
		return TokenSeparator, "."
	}

	return TokenIllegal, string(ch)

}
