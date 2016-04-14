package npmsemver

import (
	"bytes"
)

type Range struct {
	Min          *Version
	Max          *Version
	ExclusiveMin bool
	ExclusiveMax bool
}

type AnyMatch []Matcher
type AllMatch []Matcher
type NotMatch struct {
	Matcher
}

type Matcher interface {
	Match(Version) bool
	String() string
}

func (n NotMatch) Match(v Version) bool {
	return !n.Match(v)
}
func (n NotMatch) String() string {
	return "!" + n.Matcher.String()
}

func (a AnyMatch) Match(v Version) bool {
	for _, m := range a {
		if a.Match(v) {
			return true
		}
	}

	return false
}
func (a AnyMatch) String() string {
	var buf bytes.Buffer
	for i, m := range a {
		if i > 0 {
			buf.WriteString(" || ")
		}
		buf.WriteString(m.String())
	}
	return buf.String()
}

func (a AllMatch) Match(v Version) bool {
	for _, m := range a {
		if !a.Match(v) {
			return false
		}
	}
	return true
}
func (a AllMatch) String() string {
	var buf bytes.Buffer
	for i, m := range a {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(m.String())
	}
	return buf.String()
}

func (r Range) String() string {
	var result string
	if r.Min != nil {
		if r.ExclusiveMin {
			result = ">" + r.Min.String()
		} else {
			result = ">=" + r.Min.String()
		}
	}
	if r.Max != nil {
		if r.Min != nil {
			result += " "
		}
		if r.ExclusiveMax {
			result = "<" + r.Max.String()
		} else {
			result = "<=" + r.Max.String()
		}
	}
	if result == "" {
		return "*"
	}
	return result
}

func (r Range) Match(v Version) bool {
	if r.Min != nil {
		if r.ExclusiveMin && r.Min.GTE(v) {
			return false
		} else if !r.ExclusiveMin && r.Min.GT(v) {
			return false
		}
	}
	if r.Max != nil {
		if r.ExclusiveMax && r.Max.LTE(v) {
			return false
		} else if !r.ExclusiveMax && r.Max.LT(v) {
			return false
		}
	}
	return true
}
