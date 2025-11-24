package main

import (
	"slices"
)

// arithmetischer Mittelwert
func getMean(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum int64
	for _, element := range values {
		sum += element
	}

	return float64(sum) / float64(len(values))
}

// Median
func getMedian(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}

	copiedValues := slices.Clone(values)
	slices.Sort(copiedValues)

	// if its an uneven length, we can just take the middle value
	if len(copiedValues)%2 == 1 {
		return float64(copiedValues[(len(copiedValues)-1)/2.0])
	}

	middle := len(copiedValues) / 2
	median := float64(copiedValues[middle-1]+copiedValues[middle]) / 2.0

	return median
}
