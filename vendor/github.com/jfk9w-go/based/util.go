package based

import (
	"io"

	"github.com/go-playground/validator/v10"
)

// Validate is a default shared validator.Validate instance.
var validate = validator.New()

func Validate(value any) error {
	return validate.Struct(value)
}

// Unit represents an empty type, similar to Unit in Scala.
type Unit struct{}

// Nop is an empty function.
func Nop() {}

func Close(value any) error {
	if closer, ok := value.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
