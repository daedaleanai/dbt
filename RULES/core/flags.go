package core

import "strings"

type Flags []string

func (f Flags) String() string {
	return strings.Join(f, " ")
}
