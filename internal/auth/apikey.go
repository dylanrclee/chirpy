package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	auth_string := headers.Get("Authorization")
	if auth_string == "" {
		return "", fmt.Errorf("Authorization header does not exist")
	}

	auth_strings := strings.Split(auth_string, " ")
	if len(auth_strings) != 2 {
		return "", fmt.Errorf("Authorization header is not right length")
	}

	if auth_strings[0] != "ApiKey" {
		return "", fmt.Errorf("Authorization header does not contain 'ApiKey'")
	}
	return auth_strings[1], nil
}
