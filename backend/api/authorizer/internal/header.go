package internal

import (
	"errors"
	"strings"
)

func GetHeader(headers map[string]string, desiredKey string) (string, error) {
	for headerKey, headerValue := range headers {
		if strings.EqualFold(headerKey, desiredKey) {
			return headerValue, nil
		}
	}
	return "", errors.New("error: desired header '" + desiredKey + "' not found")
}
