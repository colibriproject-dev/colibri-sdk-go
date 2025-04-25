package validator

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestSortDirectionValidation(t *testing.T) {
	type testData struct {
		Direction types.SortDirection
		Expected  bool
	}

	tests := []testData{
		{Direction: types.SortDirection("ASC"), Expected: true},
		{Direction: types.SortDirection("DESC"), Expected: true},
		{Direction: types.SortDirection("asc"), Expected: false},
		{Direction: types.SortDirection("desc"), Expected: false},
		{Direction: types.SortDirection("invalid"), Expected: false},
	}

	validate := validator.New()
	validate.RegisterValidation("sort-direction", sortDirectionValidation)

	for _, test := range tests {
		err := validate.Var(test.Direction, "sort-direction")
		if test.Expected {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}
