package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyHMACSHA256(t *testing.T) {
	body := []byte(`{"ok":true}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !VerifyHMACSHA256(body, sig, "secret") {
		t.Fatal("expected valid signature")
	}
	if VerifyHMACSHA256(body, sig, "wrong") {
		t.Fatal("expected invalid signature")
	}
}
