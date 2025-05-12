package types

// SortDirection is the field sort direction
type SortDirection string

const (
	ASC  SortDirection = "ASC"
	DESC SortDirection = "DESC"
)

// sortDirections is a map that represents the valid sort directions.
// It maps a SortDirection value to a boolean indicating whether the direction is valid.
var sortDirections = map[SortDirection]bool{
	ASC:  true,
	DESC: true,
}

// IsValid checks if the SortDirection is valid.
// It returns true if the SortDirection is valid, otherwise false.
func (sd SortDirection) IsValid() bool {
	return sortDirections[sd]
}

// Sort is the contract to sort a field
type Sort struct {
	Direction SortDirection
	Field     string
}

// NewSort returns a new Sort
func NewSort(direction SortDirection, field string) Sort {
	return Sort{direction, field}
}
