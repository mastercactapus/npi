package npmsemver

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser_parseVersion(t *testing.T) {
	check := func(version string, major, minor, patch int, pre []string, build []string) {
		t.Log(version)
		p := newParser(bytes.NewBufferString(version))
		v, err := p.parseVersion()
		assert.Nil(t, err)

		assert.Equal(t, major, v.Major, "major version")
		assert.Equal(t, minor, v.Minor, "minor version")
		assert.Equal(t, patch, v.Patch, "patch version")

		assert.EqualValues(t, pre, v.Prerelease, "prerelease identifiers")
		assert.EqualValues(t, build, v.Build, "build identifiers")
	}

	check("1.2.3", 1, 2, 3, nil, nil)
	check("1.0.0", 1, 0, 0, nil, nil)
	check("1.2654.0", 1, 2654, 0, nil, nil)
	check("1.0.3", 1, 0, 3, nil, nil)

	check("1.0.0-foo", 1, 0, 0, []string{"foo"}, nil)
	check("1.0.0-foo-bar", 1, 0, 0, []string{"foo-bar"}, nil)
	check("1.0.0-foo-bar.baz", 1, 0, 0, []string{"foo-bar", "baz"}, nil)

	check("1.0.0-foo+bin.1.foo", 1, 0, 0, []string{"foo"}, []string{"bin", "1", "foo"})
	check("1.0.0-foo-bar+bin.do", 1, 0, 0, []string{"foo-bar"}, []string{"bin", "do"})
	check("1.0.0-foo-bar.baz.3+bin.1", 1, 0, 0, []string{"foo-bar", "baz", "3"}, []string{"bin", "1"})

	check("1.0.0+bin-baz.1", 1, 0, 0, nil, []string{"bin-baz", "1"})

}
