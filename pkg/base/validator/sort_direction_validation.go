package validator

import (
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
	"github.com/go-playground/validator/v10"
)

func sortDirectionValidation(fl validator.FieldLevel) bool {
	direction, _ := fl.Field().Interface().(types.SortDirection)
	return direction.IsValid()
}
