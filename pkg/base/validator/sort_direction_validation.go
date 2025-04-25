package validator

import (
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
	"github.com/go-playground/validator/v10"
)

func sortDirectionValidation(fl validator.FieldLevel) bool {
	direction, ok := fl.Field().Interface().(types.SortDirection)
	if !ok {
		return false
	}

	return direction.IsValid()
}
