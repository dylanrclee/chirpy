package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	cur_time := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(cur_time),
		ExpiresAt: jwt.NewNumericDate(cur_time.Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {

	var claims jwt.RegisteredClaims
	keyfunction := func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &claims, keyfunction)
	if err != nil {
		return uuid.Nil, err
	}

	id_string, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	parsed_id, err := uuid.Parse(id_string)
	if err != nil {
		return uuid.Nil, err
	}

	return parsed_id, nil
}
