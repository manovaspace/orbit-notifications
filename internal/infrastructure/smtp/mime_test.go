package smtp

import (
	"strings"
	"testing"
)

func TestBuildMIME_multipart(t *testing.T) {
	msg := string(BuildMIME("from@x", "to@x", "Your Manova login code", "text-body", "<p>html</p>"))
	if !strings.Contains(msg, "multipart/alternative") {
		t.Fatal(msg)
	}
	if !strings.Contains(msg, "text/plain") || !strings.Contains(msg, "text/html") {
		t.Fatal(msg)
	}
	if !strings.Contains(msg, "text-body") || !strings.Contains(msg, "<p>html</p>") {
		t.Fatal(msg)
	}
	if strings.Contains(msg, "123456") && strings.Contains(msg, "Subject:") {
		// subject in this fixture has no code
	}
}
