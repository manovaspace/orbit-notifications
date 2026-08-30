package templates

import (
	"strings"
	"testing"
)

func TestRender_otp_login(t *testing.T) {
	res, err := Render("otp_login", map[string]string{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Subject != "Your Manova login code" || !strings.Contains(res.Text, "123456") || res.HTML == "" {
		t.Fatalf("%q %q %d", res.Subject, res.Text, len(res.HTML))
	}
}

func TestRender_otp_login_sms(t *testing.T) {
	res, err := Render("otp_login_sms", map[string]string{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Subject == "" || res.Text == "" {
		t.Fatal("empty output")
	}
}

func TestRender_unknown(t *testing.T) {
	_, err := Render("nope", map[string]string{})
	if err == nil {
		t.Fatal("expected error")
	}
}
