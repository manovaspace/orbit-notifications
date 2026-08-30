package mailtemplates

import (
	"bytes"
	"strings"
	"testing"
)

func TestRender_otp_login(t *testing.T) {
	res, err := Render("otp_login", map[string]string{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Subject != "Your Manova login code" {
		t.Fatalf("subject = %q", res.Subject)
	}
	if strings.Contains(res.Subject, "123456") {
		t.Fatal("subject leaked code")
	}
	if !strings.Contains(res.Text, "123456") || !strings.Contains(res.Text, "10 minutes") {
		t.Fatalf("text = %s", res.Text)
	}
}

func TestRender_otp_login_sms(t *testing.T) {
	res, err := Render("otp_login_sms", map[string]string{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Subject != "Login code" {
		t.Fatalf("subject = %q", res.Subject)
	}
	if res.HTML != "" {
		t.Fatalf("SMS must have empty HTML, got %q", res.HTML)
	}
	if !strings.Contains(res.Text, "123456") {
		t.Fatal("sms text missing code")
	}
}

func TestRender_missing_code(t *testing.T) {
	_, err := Render("otp_login", map[string]string{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRender_unknown(t *testing.T) {
	_, err := Render("not_a_template", map[string]string{"code": "1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRender_invite_and_owner_subjects(t *testing.T) {
	inv, err := Render("invite_developer", map[string]string{
		"token":        "manova-inv.aaa.bbb",
		"cli_command":  "orbit onboard --token manova-inv.aaa.bbb",
		"curl_command": "curl -fsSL https://orbit.manova.space | bash -s -- onboard --token manova-inv.aaa.bbb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Subject != "Welcome to Manova — developer workspace invitation" {
		t.Fatalf("invite subject = %q", inv.Subject)
	}
	if strings.Contains(inv.Subject, "manova-inv") {
		t.Fatal("invite subject leaked token")
	}
	own, err := Render("owner_challenge", map[string]string{
		"code":  "749102",
		"email": "sara@manova.space",
	})
	if err != nil {
		t.Fatal(err)
	}
	if own.Subject != "Orbit — server ownership verification" {
		t.Fatalf("owner subject = %q", own.Subject)
	}
	if strings.Contains(own.Subject, "749102") {
		t.Fatal("owner subject leaked code")
	}
}

func TestRender_otp_login_html_letter(t *testing.T) {
	res, err := Render("otp_login", map[string]string{"code": "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if res.HTML == "" {
		t.Fatal("html empty")
	}
	if !strings.Contains(res.HTML, "123456") {
		t.Fatal("html missing code")
	}
	if !strings.Contains(res.HTML, "#fafafa") || !strings.Contains(res.HTML, "#2563eb") {
		t.Fatal("html missing token hex")
	}
	if strings.Contains(res.HTML, "#09090b") || strings.Contains(res.HTML, "display:flex") || strings.Contains(res.HTML, "display: flex") {
		t.Fatal("html still dark/flex")
	}
	if !strings.Contains(res.HTML, "{{.code}}") {
		// after execute the action is gone; assert executed code instead (already did)
	}
	if !strings.Contains(res.HTML, "role=\"presentation\"") && !strings.Contains(res.HTML, "role='presentation'") {
		t.Fatal("expected table presentation role from MJML")
	}
}

func TestCompiledHTMLKeepsGoActions(t *testing.T) {
	raw, err := compiled.ReadFile("compiled/otp_login.html")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("{{.code}}")) {
		t.Fatalf("compile stripped {{.code}}: %s", raw)
	}
}

func TestRender_invite_html_command_hero(t *testing.T) {
	token := "manova-inv.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.sigpad"
	res, err := Render("invite_developer", map[string]string{
		"name":         "Alex Smith",
		"token":        token,
		"cli_command":  "orbit onboard --token " + token,
		"curl_command": "curl -fsSL https://orbit.manova.space | bash -s -- onboard --token " + token,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, "orbit onboard") {
		t.Fatal("html missing command")
	}
	if strings.Contains(res.HTML, "font-size:32px") && strings.Contains(res.HTML[strings.Index(res.HTML, token):], "letter-spacing") {
		// crude: token must not appear in a 32px letter-spaced node
	}
	idx := strings.Index(res.HTML, token)
	if idx < 0 {
		t.Fatal("html missing token")
	}
	window := res.HTML[max(0, idx-200):min(len(res.HTML), idx+80)]
	if strings.Contains(window, "font-size:32px") || strings.Contains(window, "font-size:32") {
		t.Fatalf("HMAC token in 32px hero: %s", window)
	}
	if strings.Contains(res.HTML, "#09090b") {
		t.Fatal("zinc leftover")
	}
}

func TestRender_owner_html_code_hero(t *testing.T) {
	res, err := Render("owner_challenge", map[string]string{
		"code":        "749102",
		"email":       "sara@manova.space",
		"server_host": "mail.manova.space",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, "749102") || !strings.Contains(res.HTML, "sara@manova.space") {
		t.Fatal(res.HTML)
	}
	if !strings.Contains(res.HTML, "#2563eb") {
		t.Fatal("missing action hex")
	}
}
