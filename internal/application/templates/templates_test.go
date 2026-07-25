package templates

import "testing"

func TestRender_otp_login_sms(t *testing.T) {
	subject, body, err := Render("otp_login_sms", map[string]string{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if subject == "" || body == "" {
		t.Fatal("empty output")
	}
}
