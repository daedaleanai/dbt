package hdl

import (
	"sort"
	"strings"
)

func IsRtl(path string) bool {
	return strings.HasSuffix(path, ".v") ||
		strings.HasSuffix(path, ".sv") ||
		strings.HasSuffix(path, ".vhdl") ||
		strings.HasSuffix(path, ".vhd")
}

func IsVerilog(path string) bool {
	return strings.HasSuffix(path, ".v") ||
		strings.HasSuffix(path, ".sv")
}

func IsSystemVerilog(path string) bool {
	return strings.HasSuffix(path, ".sv")
}

func IsVhdl(path string) bool {
	return strings.HasSuffix(path, ".vhdl") ||
		strings.HasSuffix(path, ".vhd")
}

func IsHeader(path string) bool {
	return strings.HasSuffix(path, ".vh") ||
		strings.HasSuffix(path, ".svh") ||
		strings.HasSuffix(path, ".svp")
}

func IsConstraint(path string) bool {
	return strings.HasSuffix(path, ".xdc")
}

func IsXilinxIpCheckpoint(path string) bool {
	return strings.HasSuffix(path, ".xci")
}

func IsSimulationArchive(path string) bool {
	return strings.HasSuffix(path, ".sim.tar.gz")
}

func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
