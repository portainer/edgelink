package edgelink

import (
	"net/netip"
	"testing"
)

func TestParseDNSAddresses(t *testing.T) {
	tests := []struct {
		name     string
		dnsList  []string
		expected []netip.Addr
		wantErr  bool
	}{
		{
			name:     "Valid DNS addresses",
			dnsList:  []string{"192.168.1.1", "8.8.8.8"},
			expected: []netip.Addr{netip.MustParseAddr("192.168.1.1"), netip.MustParseAddr("8.8.8.8")},
			wantErr:  false,
		},
		{
			name:    "Invalid DNS address",
			dnsList: []string{"invalid-ip"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDNSAddresses(tt.dnsList)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDNSAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equal(got, tt.expected) {
				t.Errorf("parseDNSAddresses() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestBuildAllowedIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "Valid IP address",
			ip:       "192.168.1.1",
			expected: "192.168.1.1/30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAllowedIP(tt.ip); got != tt.expected {
				t.Errorf("buildAllowedIP() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// Helper function to compare slices of netip.Addr
func equal(a, b []netip.Addr) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
