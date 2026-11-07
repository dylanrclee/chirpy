package auth

import (
	"testing"
)

func TestHashing(t *testing.T) {
	password := "password123"
	hashed_password, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error in hashing password: %v", err)
	}

	boolean, err := CheckPasswordHash(password, hashed_password)
	if err != nil {
		t.Fatalf("Error in unhashing hash to password: %v", err)
	}

	if !boolean {
		t.Fatalf("Entered password '%v' does not equal database", password)
	}
}

func WrongPasswordTest(t *testing.T) {
	password := "password123"
	wrong_password := "asdf"

	hashed_password, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error in hashing password: %v", err)
	}

	boolean, err := CheckPasswordHash(wrong_password, hashed_password)
	if err != nil {
		t.Fatalf("Error in unhashing hash to password: %v", err)
	}

	if !boolean {
		t.Fatalf("Expected false for wrong password '%v' entered for '%s'", wrong_password, password)
	}
}
