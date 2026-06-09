package auth

import "testing"

func TestPasswordAndToken(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password to match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected wrong password not to match")
	}
	token, err := CreateToken("a-secret-that-is-long-enough", "507f1f77bcf86cd799439011")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := ParseToken("a-secret-that-is-long-enough", token)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "507f1f77bcf86cd799439011" {
		t.Fatalf("unexpected user id %q", userID)
	}
}
