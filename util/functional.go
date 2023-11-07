package util

// Maps the input slice using the provided mapping function.
func MappedSlice[V any, U any](values []V, f func(V) U) []U {
	result := make([]U, 0, len(values))
	for _, v := range values {
		result = append(result, f(v))
	}
	return result
}
