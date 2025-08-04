package validator

import (
	"github.com/go-playground/form/v4"
	playValidator "github.com/go-playground/validator/v10"
)

type Validator struct {
	validator   *playValidator.Validate
	formDecoder *form.Decoder
}

var instance *Validator

// Initialize initializes the Validator instance with playValidator and formDecoder, then registers custom types.
//
// No parameters.
// No return values.
func Initialize() {
	instance = &Validator{
		validator:   playValidator.New(),
		formDecoder: form.NewDecoder(),
	}

	registerCustomTypes()
	registerCustomValidations()
}

// RegisterCustomValidation registers a custom validation function with the provided tag.
//
// Parameters:
// - tag: the tag to be registered
// - fn: the function to be registered
// No return values.
func RegisterCustomValidation(tag string, fn playValidator.Func) {
	instance.validator.RegisterValidation(tag, fn)
}

// Struct performs validation on the provided object using the validator instance.
//
// Parameter:
// - object: the object to be validated
// Return type: error
func Struct(object any) error {
	return instance.validator.Struct(object)
}

// FormDecode decodes the values from the map[string][]string into the provided object using the formDecoder instance.
//
// Parameters:
// - object: the object to be decoded
// - values: the map containing the values to be decoded
// Return type: error
func FormDecode(object any, values map[string][]string) error {
	return instance.formDecoder.Decode(object, values)
}
