package oauthclient

import "testing"

func TestValidateIssuer(t *testing.T) {
	const issuer = "https://issuer.example.com"
	tests := []struct {
		name    string
		client  Client
		iss     string
		wantErr bool
	}{
		{"required and matching", Client{issuer: issuer, requireIssuer: true}, issuer, false},
		{"required and missing", Client{issuer: issuer, requireIssuer: true}, "", true},
		{"required and mismatched", Client{issuer: issuer, requireIssuer: true}, "https://evil.example.com", true},
		{"not required and missing", Client{issuer: issuer}, "", false},
		{"not required but mismatched", Client{issuer: issuer}, "https://evil.example.com", true},
		{"no known issuer", Client{}, "https://any.example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.ValidateIssuer(tt.iss)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIssuer(%q) error = %v; wantErr %v", tt.iss, err, tt.wantErr)
			}
		})
	}
}
