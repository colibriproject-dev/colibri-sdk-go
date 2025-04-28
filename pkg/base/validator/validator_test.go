package validator

import (
	"testing"

	playValidator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	Initialize()
	assert.NotNil(t, instance)
	assert.NotNil(t, instance.validator)
	assert.NotNil(t, instance.formDecoder)
}

func TestRegisterCustomValidation(t *testing.T) {
	Initialize()
	tag := "custom-validation"
	fn := func(fl playValidator.FieldLevel) bool {
		return true
	}

	RegisterCustomValidation(tag, fn)

	instance.validator.VarWithValue(tag, "", "")
}

func TestStruct(t *testing.T) {
	type MyStruct struct {
		Name string `validate:"required"`
	}

	Initialize()
	obj := MyStruct{Name: "John"}

	err := Struct(obj)

	assert.NoError(t, err)
}

func TestFormDecode(t *testing.T) {
	type MyForm struct {
		Email string `form:"email"`
	}

	Initialize()
	values := map[string][]string{
		"email": {"test@example.com"},
	}
	obj := MyForm{}

	err := FormDecode(&obj, values)

	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", obj.Email)
}
