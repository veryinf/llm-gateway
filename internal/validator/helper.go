package validator

import (
	"github.com/samber/lo"
)

func ToStringSlice[T ~string](slice []T) []any {
	return lo.Map(slice, func(item T, index int) any {
		return string(item)
	})
}

func ToNumberSlice[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](slice []T) []any {
	return lo.Map(slice, func(item T, index int) any {
		return float64(item)
	})
}
