package service

import "testing"

func TestValidateCartOwner(t *testing.T) {
	tests := []struct {
		name          string
		userIDPresent bool
		sessionID     string
		wantErr       bool
	}{
		{name: "user only", userIDPresent: true, sessionID: "", wantErr: false},
		{name: "session only", userIDPresent: false, sessionID: "sess_1", wantErr: false},
		{name: "both missing", userIDPresent: false, sessionID: "", wantErr: true},
		{name: "both present", userIDPresent: true, sessionID: "sess_1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCartOwner(tc.userIDPresent, tc.sessionID)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("did not expect error, got %v", err)
			}
		})
	}
}

func TestValidateOrderStatus(t *testing.T) {
	valid := []string{"pending", "paid", "failed", "refunded", "shipped"}
	for _, status := range valid {
		if err := ValidateOrderStatus(status); err != nil {
			t.Fatalf("expected status %q to be valid: %v", status, err)
		}
	}

	if err := ValidateOrderStatus("unknown"); err == nil {
		t.Fatalf("expected invalid status to return error")
	}
}

func TestValidateShippingForStatus(t *testing.T) {
	err := ValidateShippingForStatus("pending", ShippingAddress{})
	if err != nil {
		t.Fatalf("did not expect error for non-shipped status: %v", err)
	}

	err = ValidateShippingForStatus("shipped", ShippingAddress{})
	if err == nil {
		t.Fatalf("expected error when shipped without shipping fields")
	}

	addr := ShippingAddress{
		Name:        "Jane Doe",
		AddressLine: "123 Main St",
		City:        "Austin",
		PostalCode:  "78701",
		Country:     "US",
	}
	err = ValidateShippingForStatus("shipped", addr)
	if err != nil {
		t.Fatalf("expected no error for complete shipped address: %v", err)
	}
}
