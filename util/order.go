package util

import (
	"sort"

	"golang.org/x/exp/constraints"

	"github.com/daedaleanai/dbt/v2/log"
)

// OrderedMap is a map supporting iteration ordered by the key.
//
// In addition, the map aborts on an attempt to override a key. This behavior is configurable, and can be turned off.
type OrderedMap[K constraints.Ordered, V any] struct {
	data            map[K]V
	forbidOverrides bool
}

// OrderedMapEntry is an accessor into a single (key, value) pair of the map.
type OrderedMapEntry[K constraints.Ordered, V any] struct {
	Key   K
	Value V
}

// Instantiates an empty OrderedMap object.
func NewOrderedMap[K constraints.Ordered, V any]() OrderedMap[K, V] {
	return OrderedMap[K, V]{
		data:            map[K]V{},
		forbidOverrides: true,
	}
}

// Instantiates a new OrderedMap from a given conventional map
// by shallow-copying both the keys and the values.
func NewOrderedMapFrom[K constraints.Ordered, V any](raw map[K]V) OrderedMap[K, V] {
	result := OrderedMap[K, V]{
		data:            make(map[K]V, len(raw)),
		forbidOverrides: true,
	}
	for k, v := range raw {
		result.data[k] = v
	}
	return result
}

// Allow key overrides of the keys.
func (m *OrderedMap[K, V]) AllowOverrides() {
	m.forbidOverrides = false
}

// Insert a (key, value) pair.
func (m *OrderedMap[K, V]) Insert(key K, value V) {
	if m.forbidOverrides {
		if val, ok := m.data[key]; ok {
			log.Fatal(
				"Attempting to override a value with key: %v; old value: %v; new value: %v",
				key, val, value)
		}
	}
	m.data[key] = value
}

// Performs a lookup of the key, similar to `v, ok := m[k]`.
func (m *OrderedMap[K, V]) Lookup(key K) (V, bool) {
	val, ok := m.data[key]
	return val, ok
}

// Performs a lookup of the key, and aborts if the key is not found.
func (m *OrderedMap[K, V]) Get(key K) V {
	val, ok := m.Lookup(key)
	if !ok {
		log.Fatal("Could not get a value out of the map, key: %v", key)
	}
	return val
}

// Returns the list of entries ordered by keys.
func (m *OrderedMap[K, V]) Entries() []OrderedMapEntry[K, V] {
	keys := m.Keys()

	result := make([]OrderedMapEntry[K, V], 0, len(m.data))
	for _, k := range keys {
		result = append(result, OrderedMapEntry[K, V]{
			Key:   k,
			Value: m.data[k],
		})
	}
	return result
}

// Returns the ordered list of map keys.
func (m *OrderedMap[K, V]) Keys() []K {
	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// Returns the values of entries ordered by their keys.
func (m *OrderedMap[K, V]) Values() []V {
	keys := m.Keys()

	result := make([]V, 0, len(m.data))
	for _, k := range keys {
		result = append(result, m.data[k])
	}
	return result
}

// Returns the ordered copy of the provided slice, the values are shallow-copied.
func OrderedSlice[V constraints.Ordered](values []V) []V {
	result := make([]V, len(values))
	copy(result, values)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// Returns the ordered copy of the provided slice, ordering is done using the key function.
func SliceOrderedBy[V any, K constraints.Ordered](values []V, key func(v *V) K) []V {
	result := make([]V, len(values))
	copy(result, values)
	sort.Slice(result, func(i, j int) bool { return key(&result[i]) < key(&result[j]) })
	return result
}

// Convenience function, returning the list of ordered entries of the input map.
func OrderedEntries[K constraints.Ordered, V any](m map[K]V) []OrderedMapEntry[K, V] {
	tmp := NewOrderedMapFrom(m)
	return tmp.Entries()
}

// Convenience function, returning the list of ordered keys of the input map.
func OrderedKeys[K constraints.Ordered, V any](m map[K]V) []K {
	tmp := NewOrderedMapFrom(m)
	return tmp.Keys()
}

// Convenience function, returning the list of values ordered by their keys.
func OrderedValues[K constraints.Ordered, V any](m map[K]V) []V {
	tmp := NewOrderedMapFrom(m)
	return tmp.Values()
}
