package form

import (
	validate "github.com/eskermese/machinery-catalog/backend/pkg/validate"
)

const shortServiceName = "SNT" // Короткое название сервиса

// validatorDefault - валидатор по умолчанию для форм.
var validatorDefault = validate.New(shortServiceName)
