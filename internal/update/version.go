package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed semantic version. A release with a pre-release tag sorts before the same release without one.
type Version struct {
	Major, Minor, Patch int
	Pre                 string
}

// ParseVersion reads "1.2.3", "v1.2.3" and "1.2.3-rc1".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	core, pre, _ := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("%q is not a version like 1.2.3", s)
	}
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("%q is not a version like 1.2.3", s)
		}
		nums[i] = n
	}
	return Version{nums[0], nums[1], nums[2], pre}, nil
}

// String formats without a leading v.
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// Compare returns -1, 0 or 1.
func (v Version) Compare(o Version) int {
	for _, d := range [3]int{v.Major - o.Major, v.Minor - o.Minor, v.Patch - o.Patch} {
		switch {
		case d < 0:
			return -1
		case d > 0:
			return 1
		}
	}
	switch {
	case v.Pre == o.Pre:
		return 0
	case v.Pre == "":
		return 1
	case o.Pre == "":
		return -1
	case v.Pre < o.Pre:
		return -1
	}
	return 1
}
