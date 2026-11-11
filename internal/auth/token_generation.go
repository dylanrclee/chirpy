package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func MakeRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("Error in making bytes in MakeRefreshToken func: %s", err)
	}
	print(hex.EncodeToString(b))
	return hex.EncodeToString(b), nil
}
