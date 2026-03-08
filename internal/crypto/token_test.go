package crypto

import (
	"testing"
)

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestEncryptDecryptRoundtrip(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	plaintext := "123456:ABC-DEF_ghijklmnop"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if ciphertext == plaintext {
		t.Error("ciphertext should differ from plaintext")
	}

	got, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Errorf("roundtrip failed: want %q, got %q", plaintext, got)
	}
}

func TestPassthroughMode(t *testing.T) {
	enc, err := NewTokenEncryptor("")
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	if enc.IsEnabled() {
		t.Error("expected passthrough mode")
	}

	plaintext := "123456:ABC"
	ct, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ct != plaintext {
		t.Errorf("passthrough Encrypt should return plaintext, got %q", ct)
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if pt != plaintext {
		t.Errorf("passthrough Decrypt should return input, got %q", pt)
	}
}

func TestNilEncryptorPassthrough(t *testing.T) {
	var enc *TokenEncryptor

	if enc.IsEnabled() {
		t.Error("nil encryptor should not be enabled")
	}

	ct, err := enc.Encrypt("tok")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ct != "tok" {
		t.Errorf("nil encryptor Encrypt should passthrough, got %q", ct)
	}

	pt, err := enc.Decrypt("tok")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if pt != "tok" {
		t.Errorf("nil encryptor Decrypt should passthrough, got %q", pt)
	}
}

func TestDifferentCiphertexts(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	plaintext := "same-token"
	ct1, _ := enc.Encrypt(plaintext)
	ct2, _ := enc.Encrypt(plaintext)

	if ct1 == ct2 {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	enc1, _ := NewTokenEncryptor(testKey)
	// Different key (last byte changed)
	enc2, _ := NewTokenEncryptor("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdee")

	ct, _ := enc1.Encrypt("secret-token")

	_, err := enc2.Decrypt(ct)
	if err == nil {
		t.Error("decrypting with wrong key should fail")
	}
}

func TestInvalidKeyLength(t *testing.T) {
	_, err := NewTokenEncryptor("0123456789abcdef") // 16 hex chars = 8 bytes, too short
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestInvalidHexKey(t *testing.T) {
	_, err := NewTokenEncryptor("not-valid-hex-string-of-the-right-length-for-aes256-encryption!!")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}
