package internal

import (
	"fmt"
	"regexp"
)

// Source: ChatGPT
func ValidateAccountName(name string) error {

	if name == "" {
		return fmt.Errorf("no 'accountName' query parameter inputted")
	}

	if len(name) < 4 || len(name) > 32 {
		return fmt.Errorf("accountName must be between 4 and 32 characters")
	}

	// Compile REGEX for checking characters
	pattern := `^[A-Za-z0-9]+$`
	charactersRegex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("REGEX for 'accountName' did not compile")
	}

	if !charactersRegex.MatchString(name) {
		return fmt.Errorf("accountName is invalid: must only have letters and numbers")
	}

	return nil

}
