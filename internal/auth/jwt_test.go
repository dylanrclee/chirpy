package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "fdsa"
	expiresIn := time.Second * 5

	jwt_token, err := MakeJWT(userID, tokenSecret, time.Duration(expiresIn))
	if err != nil {
		t.Fatalf("Error in making JWT: %s", err)
	}

	returned_user, err := ValidateJWT(jwt_token, tokenSecret)
	if err != nil {
		t.Fatalf("User not returned: %s", err)
	}

	if returned_user != userID {
		t.Fatalf("Returned user does not equal entered user: %s", err)
	}
}

func TestWrongSecret(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "fdsa"
	wrongSecret := "wrong"
	expiresIn := time.Second * 5

	jwt_token, err := MakeJWT(userID, tokenSecret, time.Duration(expiresIn))
	if err != nil {
		t.Fatalf("Error in making JWT: %s", err)
	}

	returned_user, err := ValidateJWT(jwt_token, wrongSecret)
	if err == nil {
		t.Fatalf("Expected wrong secret error: %s", err)
	}

	if returned_user != uuid.Nil {
		t.Fatalf("Expected user to not be returned: %s", err)
	}
}

func TestExpiredToken(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "fdsa"
	expiresIn := time.Millisecond * 5

	jwt_token, err := MakeJWT(userID, tokenSecret, time.Duration(expiresIn))
	if err != nil {
		t.Fatalf("Error in making JWT: %s", err)
	}

	time.Sleep(time.Millisecond * 6)

	returned_user, err := ValidateJWT(jwt_token, tokenSecret)
	if err == nil {
		t.Fatalf("Expected token expired error: %s", err)
	}

	if returned_user == userID {
		t.Fatalf("Expected user to not be returned: %s", err)
	}
}
