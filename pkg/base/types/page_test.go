package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPageRequest(t *testing.T) {
	const (
		page uint16 = 1
		size uint16 = 10
	)

	t.Run("Should return a new PageRequest with nil sort list", func(t *testing.T) {
		s := NewPageRequest(page, size, nil)
		assert.NotNil(t, s)
		assert.Equal(t, s.Page, page)
		assert.Equal(t, s.Size, size)
		assert.Nil(t, s.Order)
		assert.Equal(t, "", s.GetOrder())
	})

	t.Run("Should return a new PageRequest with empty sort list", func(t *testing.T) {
		s := NewPageRequest(page, size, []Sort{})
		assert.NotNil(t, s)
		assert.Equal(t, s.Page, page)
		assert.Equal(t, s.Size, size)
		assert.NotNil(t, s.Order)
		assert.Equal(t, "", s.GetOrder())
	})

	t.Run("Should return a new PageRequest with one populated sort list", func(t *testing.T) {
		sort := []Sort{NewSort(ASC, "field1")}
		s := NewPageRequest(page, size, sort)
		assert.NotNil(t, s)
		assert.Equal(t, s.Page, page)
		assert.Equal(t, s.Size, size)
		assert.NotNil(t, s.Order)
		assert.Equal(t, "field1 ASC", s.GetOrder())
	})

	t.Run("Should return a new PageRequest with many populated sort list", func(t *testing.T) {
		sort := []Sort{NewSort(ASC, "field1"), NewSort(DESC, "field2")}
		s := NewPageRequest(page, size, sort)
		assert.NotNil(t, s)
		assert.Equal(t, s.Page, page)
		assert.Equal(t, s.Size, size)
		assert.NotNil(t, s.Order)
		assert.Equal(t, "field1 ASC, field2 DESC", s.GetOrder())
	})
}
