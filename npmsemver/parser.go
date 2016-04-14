package npmsemver

import (
	"fmt"
	"io"
	"strconv"
)

type parser struct {
	s   *scanner
	buf struct {
		tok token
		lit string
		n   int
	}
}

func newParser(r io.Reader) *parser {
	return &parser{s: newScanner(r)}
}

func (p *parser) scan() (tok token, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	tok, lit = p.s.Scan()
	p.buf.tok, p.buf.lit = tok, lit
	return tok, lit
}
func (p *parser) unscan() { p.buf.n = 1 }

func isOperator(tok token) bool {
	return tok == TokenLT || tok == TokenEq || tok == TokenGT || tok == TokenNot || tok == TokenTilde || tok == TokenCaret
}

func (p *parser) scanIgnoreWhitespace() (tok token, lit string) {
	tok, lit = p.scan()
	if tok == TokenWs {
		tok, lit = p.scan()
	}
	return tok, lit
}

func (p *parser) parseOperatorRange() (Matcher, error) {
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
		if err != nil {
			return nil, err
		}
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

// parseVersion parses a strict/absolute semver-version
func (p *parser) parseVersion() (v *Version, err error) {
	tok, lit := p.scanIgnoreWhitespace()
	if tok != TokenNumber {
		return nil, p.unexpTokErr(tok, lit)
	}
	v = new(Version)
	v.Major = mustParseInt(lit)
	if tok, lit = p.scan(); tok != TokenSeparator {
		return nil, p.unexpTokErr(tok, lit)
	}
	if tok, lit = p.scan(); tok != TokenNumber {
		return nil, p.unexpTokErr(tok, lit)
	}
	v.Minor = mustParseInt(lit)
	if tok, lit = p.scan(); tok != TokenSeparator {
		return nil, p.unexpTokErr(tok, lit)
	}
	if tok, lit = p.scan(); tok != TokenNumber {
		return nil, p.unexpTokErr(tok, lit)
	}
	v.Patch = mustParseInt(lit)

	tok, lit = p.scan()
	if tok != TokenBuild && tok != TokenPrerelease {
		p.unscan()
		return v, nil
	}

	if tok == TokenPrerelease {
		v.Prerelease = make([]string, 0, 5)
		for {
			tok, lit = p.scan()
			if tok != TokenIdentifier && tok != TokenNumber {
				return nil, p.unexpTokErr(tok, lit)
			}
			v.Prerelease = append(v.Prerelease, lit)
			tok, lit = p.scan()
			if tok != TokenSeparator {
				p.unscan()
				break
			}
		}
	}

	if tok == TokenBuild {
		v.Build = make([]string, 0, 5)
		for {
			tok, lit = p.scan()
			if tok != TokenIdentifier && tok != TokenNumber {
				return nil, p.unexpTokErr(tok, lit)
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
func (p *parser) parseRange() (*Range, error) {
	var min, max Version
	tok, lit := p.scanIgnoreWhitespace()
	if tok != TokenNumber && tok != TokenIdentifier {
		return nil, p.unexpTokErr(tok, lit)
	}
	if tok == TokenNumber {
		min.Major = mustParseInt(lit)
	} else if lit == "*" || lit == "x" || lit == "X" {
		// any major version = default range
		return &Range{}, nil
	} else {
		return nil, p.unexpTokErr(tok, lit)
	}

	if tok, lit = p.scan(); tok != TokenSeparator {
		p.unscan()
		max.Major = min.Major + 1
		return &Range{Min: &min, Max: &max, ExclusiveMax: true}, nil
	}

	if tok, lit = p.scan(); tok != TokenNumber && tok != TokenIdentifier {
		return nil, p.unexpTokErr(tok, lit)
	}

	if tok == TokenNumber {
		max.Major = min.Major
		min.Minor = mustParseInt(lit)
	} else if lit == "x" || lit == "X" {
		max.Major = min.Major + 1
		// any minor version
		return &Range{Min: &min, Max: &max, ExclusiveMax: true}, nil
	} else {
		return nil, p.unexpTokErr(tok, lit)
	}

	if tok, lit = p.scan(); tok != TokenSeparator {
		p.unscan()
		max.Minor = min.Minor + 1
		return &Range{Min: &min, Max: &max, ExclusiveMax: true}, nil
	}

	if tok, lit = p.scan(); tok != TokenNumber && tok != TokenIdentifier {
		return nil, p.unexpTokErr(tok, lit)
	}

	if tok == TokenNumber {
		max.Minor = min.Minor
		min.Patch = mustParseInt(lit)
		max.Patch = min.Patch
	} else if lit == "x" || lit == "X" {
		max.Minor = min.Minor + 1
		// any patch version
		return &Range{Min: &min, Max: &max, ExclusiveMax: true}, nil
	} else {
		return nil, p.unexpTokErr(tok, lit)
	}

	if tok, lit = p.scan(); tok != TokenPrerelease && tok != TokenBuild {
		p.unscan()
		// min and max should be equal
		return &Range{Min: &min, Max: &max}, nil
	}

	if tok == TokenPrerelease {
		min.Prerelease = make([]string, 0, 5)
		for {
			tok, lit = p.scan()
			if tok != TokenIdentifier && tok != TokenNumber {
				return nil, p.unexpTokErr(tok, lit)
			}
			min.Prerelease = append(min.Prerelease, lit)
			tok, lit = p.scan()
			if tok != TokenSeparator {
				p.unscan()
				break
			}
		}
	}

	if tok == TokenBuild {
		// we don't actually need the buil/metadata, but we need to consume the tokens
		for {
			tok, lit = p.scan()
			if tok != TokenIdentifier && tok != TokenNumber {
				return nil, p.unexpTokErr(tok, lit)
			}

			tok, lit = p.scan()
			if tok != TokenSeparator {
				p.unscan()
				break
			}
		}
	}

	return &Range{Min: &min, Max: &max}, nil
}

func (p *parser) unexpTokErr(tok token, lit string) error {
	// TODO include character number, tokens, etc...
	return fmt.Errorf("unexpected token '%s'", lit)
}

func (p *parser) Parse() (Matcher, error) {
	andMatch := make([]Matcher, 0, 5)
	orMatch := make([]Matcher, 0, 5)

	var m Matcher
	var err error

	var tok token
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
