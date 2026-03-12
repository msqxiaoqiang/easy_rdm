package utils

import (
	"testing"
)

func TestInitCrypto(t *testing.T) {
	InitCrypto("test-seed-123")
	if len(aesKey) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(aesKey))
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	InitCrypto("test-seed-456")

	cases := []string{
		"hello",
		"",
		"password123!@#",
		"中文密码",
		"a very long password that exceeds typical lengths and contains special chars: !@#$%^&*()",
	}

	for _, plaintext := range cases {
		encrypted, err := Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%q) error: %v", plaintext, err)
		}
		if plaintext != "" && encrypted == plaintext {
			t.Fatalf("Encrypt(%q) returned plaintext unchanged", plaintext)
		}

		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt error: %v", err)
		}
		if decrypted != plaintext {
			t.Fatalf("expected %q, got %q", plaintext, decrypted)
		}
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	InitCrypto("test-seed-789")

	a, _ := Encrypt("same-input")
	b, _ := Encrypt("same-input")
	if a == b {
		t.Fatal("two encryptions of same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	InitCrypto("test-seed")
	_, err := Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptTooShort(t *testing.T) {
	InitCrypto("test-seed")
	_, err := Decrypt("AQID") // 3 bytes, shorter than nonce
	if err == nil {
		t.Fatal("expected error for ciphertext too short")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	InitCrypto("key-a")
	encrypted, _ := Encrypt("secret")

	InitCrypto("key-b")
	_, err := Decrypt(encrypted)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestEncryptWithoutInit(t *testing.T) {
	aesKey = nil
	_, err := Encrypt("test")
	if err == nil {
		t.Fatal("expected error when crypto not initialized")
	}
}

func TestDecryptWithoutInit(t *testing.T) {
	aesKey = nil
	_, err := Decrypt("dGVzdA==")
	if err == nil {
		t.Fatal("expected error when crypto not initialized")
	}
}
