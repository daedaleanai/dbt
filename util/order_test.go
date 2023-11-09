package util

import (
	"os"
	"os/exec"
	"testing"
)

// Instantiates an empty OrderedMap object.
func TestOrderedMap(t *testing.T) {
	m := NewOrderedMap[int, string]()
	m.Insert(4, "some")
	m.Insert(5, "value")
	m.Insert(-4, "added")

	expected := []OrderedMapEntry[int, string]{
		{Key: -4, Value: "added"},
		{Key: 4, Value: "some"},
		{Key: 5, Value: "value"},
	}

	entries := m.Entries()
	keys := m.Keys()
	values := m.Values()
	if len(entries) != len(expected) {
		t.Fatal("unexpected number of entries")
	}
	if len(keys) != len(expected) {
		t.Fatal("unexpected number of keys")
	}
	if len(values) != len(expected) {
		t.Fatal("unexpected number of values")
	}
	for i := range entries {
		if entries[i] != expected[i] {
			t.Fatalf("unexpected entry at index %d", i)
		}
		if keys[i] != expected[i].Key {
			t.Fatalf("unexpected key at index %d", i)
		}
		if values[i] != expected[i].Value {
			t.Fatalf("unexpected value at index %d", i)
		}
	}
}

func TestOrderedMapFrom(t *testing.T) {
	r := map[int]string{-4: "wow", -5: "this", 10: "aint", 3: "gonna", 12: "fail"}
	m := NewOrderedMapFrom(r)
	m.Insert(9, "wanna")

	expected := []OrderedMapEntry[int, string]{
		{Key: -5, Value: "this"},
		{Key: -4, Value: "wow"},
		{Key: 3, Value: "gonna"},
		{Key: 9, Value: "wanna"},
		{Key: 10, Value: "aint"},
		{Key: 12, Value: "fail"},
	}

	entries := m.Entries()
	keys := m.Keys()
	values := m.Values()
	if len(entries) != len(expected) {
		t.Fatal("unexpected number of entries")
	}
	if len(keys) != len(expected) {
		t.Fatal("unexpected number of keys")
	}
	if len(values) != len(expected) {
		t.Fatal("unexpected number of values")
	}
	for i := range entries {
		if entries[i] != expected[i] {
			t.Fatalf("unexpected entry at index %d", i)
		}
		if keys[i] != expected[i].Key {
			t.Fatalf("unexpected key at index %d", i)
		}
		if values[i] != expected[i].Value {
			t.Fatalf("unexpected value at index %d", i)
		}
	}
}

func TestOverridesForbidden(t *testing.T) {
	if os.Getenv("CHILD") == "1" {
		m := NewOrderedMap[int, string]()
		m.Insert(1, "hello")
		m.Insert(1, "world")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestOverridesForbidden")
	cmd.Env = append(os.Environ(), "CHILD=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("process ran with err %v, want exit status 1", err)
	}
}

func TestOverridesAllowed(t *testing.T) {
	m := NewOrderedMap[int, string]()
	m.AllowOverrides()
	m.Insert(1, "hello")
	m.Insert(1, "world")

	entries := m.Entries()
	if len(entries) != 1 {
		t.Fatal("unexpected number of entries")
	}
	if entries[0].Key != 1 {
		t.Fatal("unexpected key")
	}
	if entries[0].Value != "world" {
		t.Fatal("unexpected value")
	}
}

func TestLookups(t *testing.T) {
	r := map[int]string{-4: "wow", -5: "this", 10: "aint", 3: "gonna", 12: "fail"}
	m := NewOrderedMapFrom(r)

	_, ok := m.Lookup(17)
	if ok {
		t.Fatal("lookup should have failed")
	}

	v, ok := m.Lookup(10)
	if !ok {
		t.Fatal("lookup unexpectedly failed")
	}
	if v != "aint" {
		t.Fatal("unexpected value")
	}

	if m.Get(-5) != "this" {
		t.Fatal("unexpected value")
	}
}

func TestLookupFail(t *testing.T) {
	if os.Getenv("CHILD") == "1" {
		m := NewOrderedMap[int, string]()
		m.Get(1)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestLookupFail")
	cmd.Env = append(os.Environ(), "CHILD=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("process ran with err %v, want exit status 1", err)
	}
}

func TestOrderedSlice(t *testing.T) {
	s := []int{10, 3, 523, 77, -95}
	o := OrderedSlice(s)

	expected := []int{-95, 3, 10, 77, 523}
	if len(o) != len(expected) {
		t.Fatal("wrong size")
	}
	for i := range o {
		if o[i] != expected[i] {
			t.Fatalf("wrong element %d", i)
		}
	}
}

func TestSliceOrderedBy(t *testing.T) {
	s := []int{10, 3, 523, 77, -95}
	o := SliceOrderedBy(s, func(v *int) int { return -*v })

	expected := []int{523, 77, 10, 3, -95}
	if len(o) != len(expected) {
		t.Fatal("wrong size")
	}
	for i := range o {
		if o[i] != expected[i] {
			t.Fatalf("wrong element %d", i)
		}
	}
}
