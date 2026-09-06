package common

import "slices"

// sortedKeys returns the keys of a map in ascending order, which is what Java's
// `Collections.sort(new ArrayList<>(map.keySet()))` gives.
func sortedKeys[K ~string | ~int, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
