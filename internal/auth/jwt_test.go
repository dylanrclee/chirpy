package auth

import (
	"net/http"
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

func TestGetBearerToken(t *testing.T) {
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer 12345")

	auth_string, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("No error expected in getting authorization string: %s", err)
	}
	if auth_string != "12345" {
		t.Fatalf("Wrong authorization string returned: %s", err)
	}
}

func TestGetBearerTokenHeaderDoesNotExist(t *testing.T) {
	headers := make(http.Header)

	auth_string, err := GetBearerToken(headers)
	if err == nil {
		t.Fatalf("Error expected in getting authorization string: %s", err)
	}
	if auth_string != "" {
		t.Fatalf("Expected no authorization string to be returned: %s", err)
	}
}
