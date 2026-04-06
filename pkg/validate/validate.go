package validate

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	validate *validator.Validate
)

func Get() *validator.Validate {
	once.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
	})
	return validate
}
