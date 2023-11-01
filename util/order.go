package util

import (
	"sort"

	"golang.org/x/exp/constraints"

	"github.com/daedaleanai/dbt/log"
)

type OrderedMap[K constraints.Ordered, V any] struct {
	data            map[K]V
	forbidOverrides bool
}

type OrderedMapEntry[K constraints.Ordered, V any] struct {
	Key   K
	Value V
}

func NewOrderedMap[K constraints.Ordered, V any]() OrderedMap[K, V] {
	return OrderedMap[K, V]{
		data:            map[K]V{},
		forbidOverrides: true,
	}
}

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

func (m *OrderedMap[K, V]) AllowOverrides() *OrderedMap[K, V] {
	m.forbidOverrides = false
	return m
}

func (m *OrderedMap[K, V]) Append(key K, value V) {
	if m.forbidOverrides {
		if val, ok := m.data[key]; ok {
			log.Fatal("Attempting to override a value with key: %s; value: %s", key, val)
		}
	}
	m.data[key] = value
}

func (m *OrderedMap[K, V]) Lookup(key K) (V, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *OrderedMap[K, V]) Get(key K) V {
	val, ok := m.Lookup(key)
	if !ok {
		log.Fatal("Lookup failed, key: %s", key)
	}
	return val
}

func (m *OrderedMap[K, V]) Keys() []K {
	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

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

func OrderedSlice[V constraints.Ordered](values []V) []V {
	result := make([]V, len(values))
	copy(result, values)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func OrderedBySlice[V any, K constraints.Ordered](values []V, key func(v *V) K) []V {
	result := make([]V, len(values))
	copy(result, values)
	sort.Slice(result, func(i, j int) bool { return key(&result[i]) < key(&result[j]) })
	return result
}

func OrderedKeys[K constraints.Ordered, V any](m map[K]V) []K {
	tmp := NewOrderedMapFrom(m)
	return tmp.Keys()
}

func OrderedEntries[K constraints.Ordered, V any](m map[K]V) []OrderedMapEntry[K, V] {
	tmp := NewOrderedMapFrom(m)
	return tmp.Entries()
}

func MappedSlice[V any, U any](values []V, f func(V) U) []U {
	result := make([]U, 0, len(values))
	for _, v := range values {
		result = append(result, f(v))
	}
	return result
}
