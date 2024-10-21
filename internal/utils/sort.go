package utils

import "sort"

func SortedKeys[Value interface{}](m map[string]Value) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func SortedArrayFromMap[Value interface{}](in map[string]*Value) []*Value {
	values := make([]*Value, 0, len(in))
	for _, key := range SortedKeys(in) {
		values = append(values, in[key])
	}
	return values
}
