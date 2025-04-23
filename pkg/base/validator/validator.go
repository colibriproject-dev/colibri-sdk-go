package validator

import (
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/types"
	form "github.com/go-playground/form/v4"
	playValidator "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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

	registerUUIDCustomType()
	registerIsoDateCustomType()
	registerIsoTimeCustomType()

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

// registerUUIDCustomType registers a custom type function for UUID parsing.
//
// It takes an array of strings as input parameters and returns an any type and an error.
func registerUUIDCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return uuid.Parse(vals[0])
	}, uuid.UUID{})
}

// registerIsoDateCustomType registers a custom type function for ISO date parsing.
//
// It takes an array of strings as input parameters and returns an interface{} and an error.
func registerIsoDateCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		return types.ParseIsoDate(vals[0])
	}, types.IsoDate{})
}

// registerIsoTimeCustomType registers a custom type function for ISO time parsing.
//
// It takes an array of strings as input parameters and returns an interface{} and an error.
func registerIsoTimeCustomType() {
	instance.formDecoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		return types.ParseIsoTime(vals[0])
	}, types.IsoTime{})
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

// registerCustomValidations registers all custom validations
func registerCustomValidations() {
	RegisterCustomValidation("br-states", brStatesValidation)
	RegisterCustomValidation("cnpj", brCNPJValidation)
	RegisterCustomValidation("cpf", brCPFValidation)
	RegisterCustomValidation("br-postal-code", brPostalCodeValidation)
	RegisterCustomValidation("sort-direction", sortDirectionValidation)
}
