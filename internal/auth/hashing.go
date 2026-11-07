package auth

import (
	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hash_params := argon2id.Params{
		Memory:      10,
		Iterations:  10,
		Parallelism: 8,
		SaltLength:  16,
		KeyLength:   16,
	}
	hash_string, err := argon2id.CreateHash(password, &hash_params)
	if err != nil {
		return "", err
	}
	return hash_string, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return match, err
	}
	return match, nil
}
