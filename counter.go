package main

import "sort"

// combine maps received by workers to create single map. sort it and return top ten results.
func tenMostOccurred(input []map[string]int) map[string]int {
	unifiedMap := make(map[string]int)

	// Iterate over each map in the input slice
	for _, m := range input {
		// Iterate over each key-value pair in the map
		for key, value := range m {
			// Add the value to the total for this key
			unifiedMap[key] += value
		}
	}

	// Sort the keys based on their corresponding values
	var keys []string
	for key := range unifiedMap {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return unifiedMap[keys[i]] > unifiedMap[keys[j]]
	})

	// Create a map to store the top 10 keys with their values
	topKeys := make(map[string]int)
	for _, key := range keys {
		topKeys[key] = unifiedMap[key]
		if len(topKeys) == 10 {
			break
		}
	}
	return topKeys
}
