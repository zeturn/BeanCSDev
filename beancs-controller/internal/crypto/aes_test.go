package crypto

import "testing"

func TestAESGCMCipherRoundTrip(t *testing.T) {
	c, err := NewAESGCMCipher("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := c.EncryptString("secret")
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.DecryptString(enc)
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret" {
		t.Fatalf("got %q", got)
	}
}
