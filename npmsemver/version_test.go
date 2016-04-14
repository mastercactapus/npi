package npmsemver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	check := func(version string, major, minor, patch int, pre []string, build []string) {
		v, err := Parse(version)
		assert.Nil(t, err)

		assert.Equal(t, v.Major, major)
		assert.Equal(t, v.Minor, minor)
		assert.Equal(t, v.Patch, patch)

		assert.EqualValues(t, v.Prerelease, pre)
		assert.EqualValues(t, v.Build, build)
	}

	check("1.0.0", 1, 0, 0, nil, nil)
	check("1.2.0", 1, 2, 0, nil, nil)
	check("1.0.3", 1, 0, 3, nil, nil)
	check("1.2.3", 1, 2, 3, nil, nil)

	check("1.0.0-foo", 1, 0, 0, []string{"foo"}, nil)
	check("1.0.0-foo-bar", 1, 0, 0, []string{"foo-bar"}, nil)
	check("1.0.0-foo-bar.baz", 1, 0, 0, []string{"foo-bar", "baz"}, nil)

	check("1.0.0-foo+bin", 1, 0, 0, []string{"foo"}, []string{"bin"})
	check("1.0.0-foo-bar+bin.do", 1, 0, 0, []string{"foo-bar"}, []string{"bin", "do"})
	check("1.0.0-foo-bar.baz.3+bin.1", 1, 0, 0, []string{"foo-bar", "baz"}, []string{"bin", "1"})

	check("1.0.0+bin-baz.1", 1, 0, 0, nil, []string{"bin-baz", "1"})

}
