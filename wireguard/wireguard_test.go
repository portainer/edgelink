package wireguard

import (
	"encoding/base64"
	"testing"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestParsePrivateKey(t *testing.T) {
	t.Run("ValidKey", func(t *testing.T) {
		// Generate a private key for testing and encode it to base64
		privateKey, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			t.Fatalf("Failed to generate private key for test: %v", err)
		}

		encodedPrivateKey := base64.StdEncoding.EncodeToString(privateKey[:])

		decodedPrivateKey, publicKey, err := ParsePrivateKey(encodedPrivateKey)
		if err != nil {
			t.Fatalf("ParsePrivateKey() error = %v", err)
		}

		if decodedPrivateKey != privateKey {
			t.Errorf("ParsePrivateKey() private key mismatch, got = %v, want = %v", decodedPrivateKey, privateKey)
		}

		if publicKey != privateKey.PublicKey() {
			t.Errorf("ParsePrivateKey() public key mismatch, got = %v, want = %v", publicKey, privateKey.PublicKey())
		}
	})

	t.Run("InvalidBase64String", func(t *testing.T) {
		// Use an invalid base64 string
		invalidEncodedKey := "invalid-base64-key"

		_, _, err := ParsePrivateKey(invalidEncodedKey)
		if err == nil {
			t.Error("ParsePrivateKey() expected error for invalid base64 string, got nil")
		}
	})

	t.Run("ValidBase64InvalidKey", func(t *testing.T) {
		// Create a valid base64 string that decodes to an invalid key length
		// invalidKeyBytes := make([]byte, wgtypes.KeyLen-1) // One byte short
		invalidKey := "invalid-key"
		validBase64InvalidKey := base64.StdEncoding.EncodeToString([]byte(invalidKey))

		_, _, err := ParsePrivateKey(validBase64InvalidKey)
		if err == nil {
			t.Error("ParsePrivateKey() expected error for valid base64 but invalid key, got nil")
		}
	})
}
