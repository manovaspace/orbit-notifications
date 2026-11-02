package domain

import "testing"

func TestHashRecipientIsStableSHA256(t *testing.T) {
	// SHA-256("person@example.com")
	const want = "542d240129883c019e106e3b1b2d3f3cb3537c43c425364de8e951d5a3083345"
	got := HashRecipient("person@example.com")
	if len(got) != 64 {
		t.Fatalf("hash length = %d, want 64", len(got))
	}
	if got != want {
		t.Fatalf("HashRecipient = %q, want %q", got, want)
	}
	if got == HashRecipient("other@example.com") {
		t.Fatal("different recipients produced the same hash")
	}
}
