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

// Initialize initializes the global Validator instance and registers custom types and validations.
func Initialize() {
	instance = &Validator{
		validator:   playValidator.New(),
		formDecoder: form.NewDecoder(),
	}

	registerCustomTypes()
	registerCustomValidations()
}

// RegisterCustomValidation registers a custom validation function for the given tag.
func RegisterCustomValidation(tag string, fn playValidator.Func) {
	instance.validator.RegisterValidation(tag, fn)
}

// Struct performs validation on the provided object.
func Struct(object any) error {
	return instance.validator.Struct(object)
}

// FormDecode decodes values from a map into the provided object using the form decoder.
func FormDecode(object any, values map[string][]string) error {
	return instance.formDecoder.Decode(object, values)
}
