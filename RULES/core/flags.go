package core

import (
	"strings"
)

// Flags is a list of flags passed to a tool or command.
type Flags []string

func (f Flags) String() string {
	return strings.Join(f, " ")
}
