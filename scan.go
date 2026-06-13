package shorten

import (
	"fmt"
	"reflect"
)

// ScanRows scans all rows from rows into a slice of values of type T.
// For pointer-to-struct types, column names are mapped to struct fields using
// field names or `ino` tags.
func ScanRows[T any](rows Rows) ([]T, error) {
	var values []T

	err := ScanVisit[T](rows, func(value T, _ int) bool {
		values = append(values, value)
		return true
	})

	return values, err
}

// ScanRow scans a single row from rows into a value of type T. For pointer-to-struct
// types, column names are mapped to struct fields using field names or `ino` tags.
func ScanRow[T any](rows Rows) (T, error) {
	var value T

	err := ScanVisit[T](rows, func(item T, _ int) bool {
		value = item
		return false
	})

	return value, err
}

// ScanVisit scans rows using a visitor function that is called for each row.
// The visitor receives the scanned value and its 0-based row index. Returning false
// stops iteration. For pointer-to-struct types, column names are mapped to struct
// fields using field names or `ino` tags.
func ScanVisit[T any](rows Rows, visitor func(T, int) bool) error {
	typ := reflect.TypeFor[T]()
	if typ.Kind() == reflect.Pointer {
		elem := typ.Elem()
		if elem.Kind() != reflect.Struct {
			return fmt.Errorf("shorten: expects type %s to be pointer to struct", typ)
		}

		columns := rows.Columns()
		if len(columns) == 0 {
			return fmt.Errorf("shorten: no columns found")
		}

		var i int
		for rows.Next() {
			value, fields, err := scanStruct(elem, columns)
			if err != nil {
				return err
			}

			err = rows.Scan(fields...)
			if err != nil {
				return fmt.Errorf("shorten: scan [%d]: %w", i, err)
			}

			if !visitor(value.(T), i) {
				break
			}

			i++
		}
	} else {
		var i int
		for rows.Next() {
			var item T
			err := rows.Scan(&item)
			if err != nil {
				return fmt.Errorf("shorten: scan [%d]: %w", i, err)
			}

			if !visitor(item, i) {
				break
			}

			i++
		}
	}

	return rows.Close()
}

// ScanVisitFlat scans rows using a visitor function without unmarshaling into a type.
// Destinations (dest) are passed directly to rows.Scan for each row, allowing raw
// value scanning. The visitor receives the 0-based row index. Returning false stops iteration.
func ScanVisitFlat(rows Rows, visitor func(int) bool, dest ...any) error {
	var i int
	for rows.Next() {
		err := rows.Scan(dest...)
		if err != nil {
			return fmt.Errorf("shorten: scan [%d]: %w", i, err)
		}

		if !visitor(i) {
			break
		}

		i++
	}

	return rows.Close()
}

func scanStruct(typ reflect.Type, columns []string) (any, []any, error) {
	if typ.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("mapper: expects type %s to be struct", typ)
	}

	index := getStructMapping(typ)
	values := make([]any, 0, len(columns))
	val := reflect.New(typ).Elem()
	for _, name := range columns {
		idx, found := index[name]
		if !found {
			return nil, nil, fmt.Errorf("mapper: missing destination name %q in %s", name, typ)
		}
		field := val.FieldByIndex(idx)
		values = append(values, field.Addr().Interface())
	}
	return val.Addr().Interface(), values, nil
}
