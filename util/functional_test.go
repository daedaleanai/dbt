package util

import (
	"strconv"
	"testing"
)

func TestMappedSlice(t *testing.T) {
	r := []int{123, 44, -4}
	m := MappedSlice(r, func(v int) string { return strconv.Itoa(v) })

	expected := []string{"123", "44", "-4"}
	if len(m) != len(expected) {
		t.Fatal("unexpected result size")
	}
	for i := range m {
		if m[i] != expected[i] {
			t.Fatalf("unexpected value at index %d", i)
		}
	}
}
