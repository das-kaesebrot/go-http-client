package main

import (
	"fmt"
	"math"
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

type (
	Decimal float64
)

const (
	kb Decimal = 1e+03
	mb Decimal = 1e+06
	gb Decimal = 1e+09
	tb Decimal = 1e+12
	pb Decimal = 1e+15
	eb Decimal = 1e+18
)

const (
	precision0 = "%.0f\u00A0%s%s"
	precision1 = "%.1f\u00A0%s%s"
	precision2 = "%.2f\u00A0%s%s"
)

func (n Decimal) Bits() string {
	return n.String("b")
}

func (n Decimal) String(unit string) string {
	f := n
	x := Decimal(math.Abs(float64(n)))
	switch {
	case x >= eb:
		f /= eb
		return fmt.Sprintf(precision2, f, "E", unit)
	case x >= pb:
		f /= pb
		return fmt.Sprintf(precision2, f, "P", unit)
	case x >= tb:
		f /= tb
		return fmt.Sprintf(precision2, f, "T", unit)
	case x >= gb:
		f /= gb
		return fmt.Sprintf(precision2, f, "G", unit)
	case x >= mb:
		f /= mb
		return fmt.Sprintf(precision2, f, "M", unit)
	case x >= kb:
		f /= kb
		return fmt.Sprintf(precision1, f, "k", unit)
	default:
		return fmt.Sprintf(precision0, f, "", unit)
	}
}
