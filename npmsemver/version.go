package npmsemver

import (
	"bytes"
	"fmt"
	"strings"
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease []string
	Build      []string
}

func Parse(s string) (v Version, err error) {
	p := newParser(bytes.NewBufferString(s))
	vs, err := p.parseVersion()
	if err != nil {
		return v, err
	}
	return *vs, nil
}

func (a Version) LT(b Version) bool {
	return false
}
func (a Version) EQ(b Version) bool {
	return false

}
func (a Version) GT(b Version) bool {
	return false

}
func (a Version) LTE(b Version) bool {
	return false

}
func (a Version) GTE(b Version) bool {
	return false

}
func (a Version) Match(v Version) bool {
	return a.EQ(v)
}
func (a Version) String() string {
	var prerelease, build string
	if a.Prerelease != nil && len(a.Prerelease) > 0 {
		prerelease = "-" + strings.Join(a.Prerelease, ".")
	}
	if a.Build != nil && len(a.Build) > 0 {
		build = "+" + strings.Join(a.Build, ".")
	}
	return fmt.Sprintf("%d.%d.%d%s%s", a.Major, a.Minor, a.Patch, prerelease, build)
}
