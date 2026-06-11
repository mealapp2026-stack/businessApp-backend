package httpapi

import (
	"testing"

	"businessapp/backend/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNormalizePhone(t *testing.T) {
	tests := map[string]string{
		"+234 801 234 5678": "+2348012345678",
		"0801-234-5678":     "08012345678",
		"(0801) 234.5678":   "08012345678",
	}
	for input, expected := range tests {
		if actual := normalizePhone(input); actual != expected {
			t.Errorf("normalizePhone(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestSupportedCurrencies(t *testing.T) {
	for _, code := range []string{"NGN", "USD", "EUR", "XOF"} {
		if _, ok := supportedCurrencies[code]; !ok {
			t.Errorf("expected %s to be supported", code)
		}
	}
	if _, ok := supportedCurrencies["INVALID"]; ok {
		t.Error("unexpected invalid currency support")
	}
}

func TestNormalizeUserTreatsExistingUsersAsOwners(t *testing.T) {
	id := primitive.NewObjectID()
	user := normalizeUser(model.User{
		ID:       id,
		Business: model.BusinessProfile{Name: "Legacy Business"},
	})

	if user.AccountID != id {
		t.Fatalf("AccountID = %s, want %s", user.AccountID, id)
	}
	if user.Role != "owner" {
		t.Fatalf("Role = %q, want owner", user.Role)
	}
	if user.Name != "Legacy Business" {
		t.Fatalf("Name = %q, want Legacy Business", user.Name)
	}
}

func TestNormalizeUserPreservesStaffAccount(t *testing.T) {
	accountID := primitive.NewObjectID()
	user := normalizeUser(model.User{
		ID:        primitive.NewObjectID(),
		AccountID: accountID,
		Name:      "Ada",
		Role:      "staff",
	})

	if user.AccountID != accountID {
		t.Fatalf("AccountID = %s, want %s", user.AccountID, accountID)
	}
	if user.Role != "staff" {
		t.Fatalf("Role = %q, want staff", user.Role)
	}
}
