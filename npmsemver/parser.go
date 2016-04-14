package npmsemver

import (
	"fmt"
	"io"
	"strconv"
)

type Parser struct {
	s   *Scanner
	buf struct {
		tok Token
		lit string
		n   int
	}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{s: NewScanner(r)}
}

func (p *Parser) scan() (tok Token, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	tok, lit = p.s.Scan()
	p.buf.tok, p.buf.lit = tok, lit
	return tok, lit
}
func (p *Parser) unscan() { p.buf.n = 1 }

func isOperator(tok Token) {
	return tok == TokenLT || tok == TokenEq || tok == TokenGT || tok == TokenNot || tok == TokenTilde || tok == TokenCaret
}

func (p *Parser) scanIgnoreWhitespace() (tok Token, lit string) {
	tok, lit = p.scan()
	if tok == TokenWs {
		tok, lit = p.scan()
	}
	return tok, lit
}

func (p *Parser) parseOperatorRange() (Matcher, error) {
	tok, lit := p.scan()
	switch tok {
	case TokenNot:
		v, err := p.parseVersion()
		if err != nil {
			return nil, err
		}
		return NotMatch{v}, nil
	case TokenCaret:
		r, err := p.parseRange()
		if err != nil {
			return nil, err
		}
		// minimum is safe, maximum is determined by caret (takes priority)

		r.Max.Prerelease = nil
		if r.Min.Major > 0 {
			if r.Max.Major > r.Min.Major {
				r.Max.Major = r.Min.Major + 1
				r.ExclusiveMax = true
			}
			return r, nil
		}

		if r.Min.Minor > 0 {
			if r.Max.Minor > r.Min.Minor {
				r.Max.Minor = r.Min.Minor + 1
				r.ExclusiveMax = true
			}
			return r, nil
		}

		if r.Min.Patch > 0 {
			if r.Max.Patch > r.Min.Patch {
				r.Max.Patch = r.Min.Patch + 1
				r.ExclusiveMax = true
			}
			return r, nil
		}

	case TokenTilde: // same as Caret, but locking down to minor
		r, err := p.parseRange()
		if err != nil {
			return nil, err
		}
		// minimum is safe, maximum is determined by tilde (takes priority)

		r.Max.Prerelease = nil
		r.Max.Major = r.Min.Major

		if r.Min.Minor > 0 {
			if r.Max.Minor > r.Min.Minor {
				r.Max.Minor = r.Min.Minor + 1
				r.ExclusiveMax = true
			}
			return r, nil
		}

		if r.Min.Patch > 0 {
			if r.Max.Patch > r.Min.Patch {
				r.Max.Patch = r.Min.Patch + 1
				r.ExclusiveMax = true
			}
			return r, nil
		}
	case TokenLT, TokenGT, TokenEq:
		orig := tok
		var hasEq bool
		for {
			// swallow extra equal "=" runes
			tok, lit = p.scan()
			if tok == TokenEq {
				hasEq = true
				continue
			}
			p.unscan()
			break
		}

		r, err := p.parseRange()
		switch {
		case orig == TokenLT && hasEq: // less-than or equal-to largest value allowed by range 'r'
			// no min
			r.Min = nil
			return r, nil
		case orig == TokenLT && !hasEq: // less-than the smallest value allowed by range 'r'
			// a simple range cannot have an exclusive min
			// so we just dump the min on the max, and remove the min
			r.Max = r.Min
			r.Min = nil
			r.ExclusiveMax = true
			return r, nil
		case orig == TokenGT && hasEq: // greater-than or equal-to the smallest value allowed by range 'r'
			// no max
			r.Max = nil
			return r, nil
		case orig == TokenGT && !hasEq: // greater-than the largest value allowed by range 'r'
			r.Min = r.Max
			r.Max = nil
			r.ExclusiveMin = false
			return r, nil
		default: // equal same as the range itself
			return r, nil
		}
	}

	return nil, p.unexpTokErr(tok, lit)
}

func mustParseInt(val string) int {
	i, err := strconv.Atoi(val)
	if err != nil {
		panic(err)
	}
	return i
}

// parseVersion parses a strict-real semver-version
func (p *Parser) parseVersion() (v Version, err error) {
	tok, lit := p.scanIgnoreWhitespace()
	if tok != TokenNumber {
		return v, p.unexpTokErr(tok, lit)
	}
	v.Major = mustParseInt(lit)
	if tok, lit = p.scan(); tok != TokenSeparator {
		return v, p.unexpTokErr(tok, lit)
	}
	if tok, lit = p.scan(); tok != TokenNumber {
		return v, p.unexpTokErr(tok, lit)
	}
	v.Minor = mustParseInt(lit)
	if tok, lit = p.scan(); tok != TokenSeparator {
		return v, p.unexpTokErr(tok, lit)
	}
	if tok, lit = p.scan(); tok != TokenNumber {
		return v, p.unexpTokErr(tok, lit)
	}
	v.Patch = mustParseInt(lit)

	tok, lit = p.scan()
	if tok != TokenMetadata && tok != TokenPrerelease {
		p.unscan()
		return v, nil
	}

	if tok == TokenPrerelease {
		v.Prerelease = make([]string, 0, 5)
		for {
			tok, lit = p.scan()
			if tok != TokenIdentifier && tok != TokenNumber {
				return v, p.unexpTokErr(tok, lit)
			}
			v.Prerelease = append(v.Prerelease, lit)
			tok, lit = p.scan()
			if tok != TokenSeparator {
				p.unscan()
				break
			}
		}
	}

	if tok == TokenMetadata {
		v.Build = make([]string, 0, 5)
		for {
			tok, lit = p.scan()
			if tok != TokenIdentifier && tok != TokenNumber {
				return v, p.unexpTokErr(tok, lit)
			}
			v.Build = append(v.Build, lit)
			tok, lit = p.scan()
			if tok != TokenSeparator {
				p.unscan()
				break
			}
		}
	}
	return v, nil
}

// parseRange will sort-of parse semver, but turns it into a range (e.g. "1" "1.x" etc)
func (p *Parser) parseRange() (Range, error) {

}

func (p *Parser) err(err error) error {
	// TODO include character number, tokens, etc...
	return err
}
func (p *Parser) unexpTokErr(tok Token, lit string) error {

}

func (p *Parser) Parse() (Matcher, error) {
	andMatch := make([]Matcher, 0, 5)
	orMatch := make([]Matcher, 0, 5)

	var m Matcher
	var err error

	var tok Token
	var lit string

	for {
		tok, lit = p.scanIgnoreWhitespace()
		if tok == TokenEOF {
			if len(andMatch) > 0 {
				orMatch = append(orMatch, AllMatch(andMatch))
			}
			break
		}

		if isOperator(tok) {
			p.unscan()
			m, err = p.parseOperatorRange()
			if err != nil {
				return nil, fmt.Errorf("parse: %s", err.Error())
			}
			andMatch = append(andMatch, m)
			continue
		}

		if tok == TokenIdentifier || tok == TokenNumber {
			p.unscan()
			m, err = p.parseRange()
			if err != nil {
				return nil, err
			}
			andMatch = append(andMatch, m)
			continue
		}

		if tok == TokenOr {
			if len(andMatch) == 0 {
				// short-circuit -- "||anthing" is the same as "*"
				return Range{}, nil
			}
			orMatch = append(orMatch, AllMatch(andMatch))
			andMatch = andMatch[:0]
			continue
		}

		return nil, p.unexpTokErr(tok, lit)
	}

	if len(orMatch) == 1 {
		return orMatch[0], nil
	} else if len(orMatch) == 0 {
		return Range{}, nil
	}
	return AnyMatch(orMatch), nil
}
