package validator

// registerCustomValidations registers all custom validations
func registerCustomValidations() {
	RegisterCustomValidation("br-states", brStatesValidation)
	RegisterCustomValidation("cnpj", brCNPJValidation)
	RegisterCustomValidation("cpf", brCPFValidation)
	RegisterCustomValidation("br-postal-code", brPostalCodeValidation)
	RegisterCustomValidation("sort-direction", sortDirectionValidation)
}
