package core

import (
	"fmt"
	"reflect"

	"github.com/samber/lo"
)

type UpdateEntry[T any] struct {
	Original T
	Input    T
}

func CurdCompare[T any](originals []T, inputs []T, compare func(a, b T) bool) (add []T, update []UpdateEntry[T], delete []T) {
	for _, original := range originals {
		input, found := lo.Find(inputs, func(input T) bool { return compare(original, input) })
		if found {
			update = append(update, UpdateEntry[T]{Original: original, Input: input})
		} else {
			delete = append(delete, original)
		}
	}
	for _, input := range inputs {
		_, found := lo.Find(originals, func(original T) bool { return compare(original, input) })
		if !found {
			add = append(add, input)
		}
	}
	return
}

func copyPropertiesWithInvoker[T any](source T, target T, invoker func(source reflect.Value, target reflect.Value) error, properties ...string) error {
	vs := reflect.ValueOf(source)
	vt := reflect.ValueOf(target)
	if vs.Kind() != reflect.Ptr || vt.Kind() != reflect.Ptr {
		return fmt.Errorf("source and target must be pointers to structs")
	}
	sVal := vs.Elem()
	tVal := vt.Elem()
	if sVal.Kind() != reflect.Struct || tVal.Kind() != reflect.Struct {
		return fmt.Errorf("source and target must  point to struct types")
	}
	for _, property := range properties {
		sField := sVal.FieldByName(property)
		tField := tVal.FieldByName(property)
		if !sField.IsValid() {
			return fmt.Errorf("source does not have field: %s", property)
		}
		if !tField.IsValid() {
			return fmt.Errorf("target does not have field: %s", property)
		}
		if !sField.CanInterface() {
			return fmt.Errorf("cannot read from source field: %s", property)
		}
		if !tField.CanSet() {
			return fmt.Errorf("cannot set target field: %s", property)
		}
		err := invoker(sField, tField)
		if err != nil {
			return err
		}
	}
	return nil
}

func CopyProperties[T any](source T, target T, properties ...string) error {
	return copyPropertiesWithInvoker(source, target, func(source reflect.Value, target reflect.Value) error {
		target.Set(source)
		return nil
	}, properties...)
}

func CopyPropertiesNotEmpty[T any](source T, target T, properties ...string) error {
	return copyPropertiesWithInvoker(source, target, func(source reflect.Value, target reflect.Value) error {
		//if source.Type().Kind() == reflect.String && source.String() == "" {
		//	return nil
		//}
		if !source.IsZero() {
			target.Set(source)
		}
		return nil
	}, properties...)
}

func FileSize(size int64) string {
	if size <= 0 {
		return "0 B"
	}

	const (
		unitKB = 1024.0
		unitMB = 1024.0 * 1024
		unitGB = 1024.0 * 1024 * 1024
		unitTB = 1024.0 * 1024 * 1024 * 1024
	)

	switch {
	case float64(size) < unitKB:
		return fmt.Sprintf("%d B", size)
	case float64(size) < unitMB:
		return fmt.Sprintf("%.2f KB", float64(size)/unitKB)
	case float64(size) < unitGB:
		return fmt.Sprintf("%.2f MB", float64(size)/unitMB)
	case float64(size) < unitTB:
		return fmt.Sprintf("%.2f GB", float64(size)/unitGB)
	default:
		return fmt.Sprintf("%.2f TB", float64(size)/unitTB)
	}
}
