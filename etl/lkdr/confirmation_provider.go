package lkdr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

type stdinConfirmationProvider struct{}

func (p stdinConfirmationProvider) GetConfirmationCode(ctx context.Context, phone string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter confirmation code for %s: ", phone)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.Wrap(err, "read line from stdin")
	}

	return strings.Trim(text, " \n\t\v"), nil
}
