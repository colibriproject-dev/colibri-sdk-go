package sqlDB

import (
	"database/sql"
	"reflect"
	"strings"

	"github.com/lib/pq"
	"golang.org/x/exp/slices"
)

// getDataList retrieves a list of items from the given sql.Rows object.
//
// It takes a sql.Rows object as input and returns a list of items type T and an error.
func getDataList[T any](rows *sql.Rows) ([]T, error) {
	list := make([]T, 0)
	for rows.Next() {
		model := new(T)
		err := rows.Scan(reflectCols(model)...)
		if err != nil {
			return nil, err
		}

		list = append(list, *model)
	}

	return list, nil
}

// reflectCols generates a list of column values from the provided model.
//
// model: the model to reflect columns from
// []any: a list of column values
func reflectCols(model any) (cols []any) {
	typeOf := reflect.TypeOf(model).Elem()
	valueOf := reflect.ValueOf(model).Elem()

	isStruct, isTime, isNull, isSlice := reflectValueValidations(valueOf)
	if isSlice {
		cols = append(cols, pq.Array(valueOf.Addr().Interface()))
	} else if !isStruct || isTime || isNull {
		cols = append(cols, valueOf.Addr().Interface())
		return
	}

	for i := 0; i < typeOf.NumField(); i++ {
		field := valueOf.Field(i)

		isStruct, isTime, isNull, isSlice = reflectValueValidations(field)
		if isSlice {
			cols = append(cols, pq.Array(field.Addr().Interface()))
		} else if isStruct && !isTime && !isNull {
			cols = append(cols, reflectCols(field.Addr().Interface())...)
		} else {
			cols = append(cols, field.Addr().Interface())
		}
	}

	return cols
}

// reflectValueValidations validates the type of the provided value.
//
// value: the value to validate
// (isStruct, isTime, isNull, isSlice): returns booleans indicating if the value is a struct, time type, null type, or a slice.
func reflectValueValidations(value reflect.Value) (isStruct, isTime, isNull, isSlice bool) {
	isStruct = value.Kind() == reflect.Struct
	isTime = slices.Contains([]string{"time.Time", "types.IsoDate", "types.IsoTime"}, value.Type().String())
	isNull = strings.Contains(value.Type().String(), "Null")
	isSlice = value.Kind() == reflect.Slice
	return
}
