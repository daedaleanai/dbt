package util

import (
	"fmt"
	"regexp"
	"strconv"
)

type Version struct {
	Major uint
	Minor uint
	Patch uint
}

var DbtVersion = Version{2, 0, 0}

func New(s string) (Version, error) {
	re := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	match := re.FindStringSubmatch(s)
	if match == nil {
		return Version{}, fmt.Errorf("invalid version string")
	}

	parts := []uint{}
	for _, m := range match[1:] {
		part, err := strconv.ParseUint(m, 10, 32)
		if err != nil {
			return Version{}, err
		}
		parts = append(parts, uint(part))
	}
	return Version{parts[0], parts[1], parts[2]}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}
