package validator

import (
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
	"github.com/google/uuid"
)

// registerCustomTypes registers all custom types.
func registerCustomTypes() {
	registerUUIDCustomType()
	registerDateTimeCustomType()
	registerIsoDateCustomType()
	registerIsoTimeCustomType()
	registerMonetaryCustomType()
	registerNullBoolCustomType()
	registerNullDateTimeCustomType()
	registerNullFloat64CustomType()
	registerNullInt16CustomType()
	registerNullInt32CustomType()
	registerNullInt64CustomType()
	registerNullIsoDateCustomType()
	registerNullIsoTimeCustomType()
}

// registerUUIDCustomType registers a custom type function for UUID parsing.
//
// It takes an array of strings as input parameters and returns an any type and an error.
func registerUUIDCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return uuid.Parse(vals[0])
	}, uuid.UUID{})
}

// registerDateTimeCustomType registers a custom type function for date time parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerDateTimeCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseDateTime(vals[0])
	}, types.DateTime{})
}

// registerIsoDateCustomType registers a custom type function for ISO date parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerIsoDateCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseIsoDate(vals[0])
	}, types.IsoDate{})
}

// registerIsoTimeCustomType registers a custom type function for ISO time parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerIsoTimeCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseIsoTime(vals[0])
	}, types.IsoTime{})
}

// registerMonetaryCustomType registers a custom type function for monetary parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerMonetaryCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseMonetary(vals[0])
	}, types.Monetary{})
}

// registerNullBoolCustomType registers a custom type function for null bool parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullBoolCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullBool(vals[0])
	}, types.NullBool{})
}

// registerNullDateTimeCustomType registers a custom type function for null date time parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullDateTimeCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullDateTime(vals[0])
	}, types.NullDateTime{})
}

// registerNullFloat64CustomType registers a custom type function for null float64 parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullFloat64CustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullFloat64(vals[0])
	}, types.NullFloat64{})
}

// registerNullInt16CustomType registers a custom type function for null int16 parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullInt16CustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullInt16(vals[0])
	}, types.NullInt16{})
}

// registerNullInt32CustomType registers a custom type function for null int32 parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullInt32CustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullInt32(vals[0])
	}, types.NullInt32{})
}

// registerNullInt64CustomType registers a custom type function for null int64 parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullInt64CustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullInt64(vals[0])
	}, types.NullInt64{})
}

// registerNullIsoDateCustomType registers a custom type function for null date parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullIsoDateCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullIsoDate(vals[0])
	}, types.NullIsoDate{})
}

// registerNullIsoTimeCustomType registers a custom type function for null time parsing.
//
// It takes an array of strings as input parameters and returns an any and an error.
func registerNullIsoTimeCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return types.ParseNullIsoTime(vals[0])
	}, types.NullIsoTime{})
}
