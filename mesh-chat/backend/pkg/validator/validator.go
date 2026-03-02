package validator

import (
	"fmt"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *playground.Validate
}

func New() *Validator {
	return &Validator{
		validate: playground.New(),
	}
}

func (v *Validator) Struct(value any) error {
	if err := v.validate.Struct(value); err != nil {
		validationErrors, ok := err.(playground.ValidationErrors)
		if !ok {
			return err
		}

		messages := make([]string, 0, len(validationErrors))
		for _, validationErr := range validationErrors {
			messages = append(messages, fmt.Sprintf("%s failed on %s", strings.ToLower(validationErr.Field()), validationErr.Tag()))
		}

		return fmt.Errorf("validation error: %s", strings.Join(messages, ", "))
	}

	return nil
}
