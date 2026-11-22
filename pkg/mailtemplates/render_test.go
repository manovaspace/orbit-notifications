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
	// after execute the action is gone; assert executed code instead (already did)
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
	token := "manova-inv.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.sigpad" // gitleaks:allow
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

func orgInviteVars(locale string) map[string]string {
	return map[string]string{
		"locale":       locale,
		"org_name":     "Acme Labs",
		"inviter_name": "Sara Ahmadi",
		"accept_token": "inv_8f3a2b",
		"expires_at":   "2026-11-09 12:00 UTC",
	}
}

func TestRender_org_invite_locales(t *testing.T) {
	tests := []struct {
		locale  string
		subject string
		want    []string
		rtl     bool
	}{
		{
			locale:  "fa",
			subject: "دعوت به سازمان — Manova",
			want: []string{
				"Sara Ahmadi شما را به سازمان Acme Labs دعوت کرده است.",
				"کد دعوت:",
				"inv_8f3a2b",
				"اعتبار دعوت تا: 2026-11-09 12:00 UTC",
				"برنامه Manova",
			},
			rtl: true,
		},
		{
			locale:  "en",
			subject: "You have been invited to an organization — Manova",
			want: []string{
				"Sara Ahmadi invited you to join Acme Labs.",
				"Invite code:",
				"inv_8f3a2b",
				"Paste this code in the Manova app to accept the invitation.",
				"This invitation expires at: 2026-11-09 12:00 UTC",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.locale, func(t *testing.T) {
			res, err := Render("org_invite", orgInviteVars(tc.locale))
			if err != nil {
				t.Fatal(err)
			}
			if res.Subject != tc.subject {
				t.Fatalf("subject = %q, want %q", res.Subject, tc.subject)
			}
			if strings.Contains(res.Subject, "inv_8f3a2b") {
				t.Fatal("subject leaked invite token")
			}
			for _, want := range tc.want {
				if !strings.Contains(res.Text, want) {
					t.Fatalf("text missing %q:\n%s", want, res.Text)
				}
				if !strings.Contains(res.HTML, want) {
					t.Fatalf("html missing %q", want)
				}
			}
			if tc.rtl {
				if !strings.Contains(res.HTML, `dir="rtl"`) || !strings.Contains(res.HTML, "direction:rtl") {
					t.Fatal("fa html missing RTL marker")
				}
			} else if !strings.Contains(res.HTML, `lang="en" dir="ltr"`) {
				t.Fatal("en html missing LTR marker")
			}
		})
	}
}

func TestRender_org_invite_defaults_to_fa(t *testing.T) {
	vars := orgInviteVars("")
	delete(vars, "locale")

	res, err := Render("org_invite", vars)
	if err != nil {
		t.Fatal(err)
	}
	if res.Subject != "دعوت به سازمان — Manova" {
		t.Fatalf("default subject = %q", res.Subject)
	}
	if !strings.Contains(res.Text, "کد دعوت") || !strings.Contains(res.HTML, `dir="rtl"`) {
		t.Fatal("default locale did not render Persian")
	}
}

func TestRender_org_invite_escapes_html(t *testing.T) {
	vars := orgInviteVars("en")
	vars["org_name"] = `<script>alert("x")</script>`

	res, err := Render("org_invite", vars)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.HTML, "<script>") {
		t.Fatalf("html contains unescaped script: %s", res.HTML)
	}
	if !strings.Contains(res.HTML, "&lt;script&gt;") {
		t.Fatalf("html does not contain escaped script: %s", res.HTML)
	}
}

func TestRender_org_invite_requires_vars(t *testing.T) {
	for _, key := range []string{"org_name", "inviter_name", "accept_token", "expires_at"} {
		t.Run(key, func(t *testing.T) {
			vars := orgInviteVars("en")
			delete(vars, key)

			if _, err := Render("org_invite", vars); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("err = %v, want missing %s", err, key)
			}
		})
	}
}

func signupVerifyVars(locale string) map[string]string {
	return map[string]string{
		"locale":             locale,
		"code":               "654321",
		"expires_in_minutes": "10",
	}
}

func TestRender_signup_verify_locales(t *testing.T) {
	tests := []struct {
		locale   string
		subject  string
		wantText []string
		wantHTML []string
		rtl      bool
	}{
		{
			locale:  "fa",
			subject: "کد تأیید Manova",
			wantText: []string{
				"کد تأیید Manova: 654321",
				"برای تأیید حساب جدید، این کد را در برنامه Manova وارد کنید.",
				"این کد تا 10 دقیقه دیگر معتبر است.",
			},
			wantHTML: []string{
				"654321",
				"برای تأیید حساب جدید، این کد را در برنامه Manova وارد کنید.",
				"این کد تا 10 دقیقه دیگر معتبر است.",
			},
			rtl: true,
		},
		{
			locale:  "en",
			subject: "Your Manova verification code",
			wantText: []string{
				"Your Manova verification code is: 654321",
				"Enter this code in the Manova app to verify your new account.",
				"This code expires in 10 minutes.",
			},
			wantHTML: []string{
				"654321",
				"Enter this code in the Manova app to verify your new account.",
				"This code expires in 10 minutes.",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.locale, func(t *testing.T) {
			res, err := Render("signup_verify", signupVerifyVars(tc.locale))
			if err != nil {
				t.Fatal(err)
			}
			if res.Subject != tc.subject || strings.Contains(res.Subject, "654321") {
				t.Fatalf("subject = %q", res.Subject)
			}
			for _, want := range tc.wantText {
				if !strings.Contains(res.Text, want) {
					t.Fatalf("text missing %q:\n%s", want, res.Text)
				}
			}
			for _, want := range tc.wantHTML {
				if !strings.Contains(res.HTML, want) {
					t.Fatalf("html missing %q", want)
				}
			}
			if tc.rtl {
				if !strings.Contains(res.HTML, `dir="rtl"`) || !strings.Contains(res.HTML, "direction:rtl") {
					t.Fatal("fa html missing RTL marker")
				}
			} else if !strings.Contains(res.HTML, `lang="en" dir="ltr"`) {
				t.Fatal("en html missing LTR marker")
			}
		})
	}
}

func TestRender_signup_verify_requires_vars(t *testing.T) {
	for _, key := range []string{"code", "expires_in_minutes"} {
		t.Run(key, func(t *testing.T) {
			vars := signupVerifyVars("en")
			delete(vars, key)

			if _, err := Render("signup_verify", vars); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("err = %v, want missing %s", err, key)
			}
		})
	}
}

func TestCompiledLocalizedHTMLKeepsGoActions(t *testing.T) {
	files := map[string]string{
		"compiled/org_invite.fa.html":    "{{.org_name}}",
		"compiled/org_invite.en.html":    "{{.accept_token}}",
		"compiled/signup_verify.fa.html": "{{.code}}",
		"compiled/signup_verify.en.html": "{{.expires_in_minutes}}",
	}
	for file, action := range files {
		raw, err := compiled.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(raw, []byte(action)) {
			t.Fatalf("%s lost %s", file, action)
		}
	}
}

const orgInviteAcceptURL = "https://acme.falak.demo.manova.space/invite#inv_8f3a2b"

func TestRender_org_invite_with_accept_url(t *testing.T) {
	tests := []struct {
		locale     string
		button     string
		fallback   string
		textLinkOK bool
	}{
		{locale: "fa", button: "پذیرش دعوت", fallback: "اگر دکمه باز نشد، این نشانی را باز کنید:"},
		{locale: "en", button: "Accept the invitation", fallback: "If the button does not work, open this link:"},
	}
	for _, tc := range tests {
		t.Run(tc.locale, func(t *testing.T) {
			vars := orgInviteVars(tc.locale)
			vars["accept_url"] = orgInviteAcceptURL
			res, err := Render("org_invite", vars)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(res.HTML, `href="`+orgInviteAcceptURL+`"`) {
				t.Fatalf("html missing unescaped href with fragment: %s", res.HTML)
			}
			if !strings.Contains(res.HTML, tc.button) {
				t.Fatalf("html missing button %q", tc.button)
			}
			if !strings.Contains(res.HTML, tc.fallback) {
				t.Fatalf("html missing fallback line %q", tc.fallback)
			}
			if !strings.Contains(res.HTML, `<span dir="ltr">`+orgInviteAcceptURL+`</span>`) {
				t.Fatal("html missing ltr link text")
			}
			if !strings.Contains(res.HTML, "inv_8f3a2b") {
				t.Fatal("html missing code fallback")
			}
			if !strings.Contains(res.Text, orgInviteAcceptURL) {
				t.Fatalf("text missing accept url:\n%s", res.Text)
			}
			if strings.Contains(res.HTML, "<no value>") || strings.Contains(res.Text, "<no value>") {
				t.Fatal("rendered <no value>")
			}
		})
	}
}

func TestRender_org_invite_without_accept_url(t *testing.T) {
	for _, locale := range []string{"fa", "en"} {
		t.Run(locale, func(t *testing.T) {
			res, err := Render("org_invite", orgInviteVars(locale))
			if err != nil {
				t.Fatal(err)
			}
			for _, banned := range []string{"پذیرش دعوت", "Accept the invitation", "open this link", "نشانی را باز کنید", "/invite#", "<no value>"} {
				if strings.Contains(res.HTML, banned) || strings.Contains(res.Text, banned) {
					t.Fatalf("output without accept_url contains %q", banned)
				}
			}
			if !strings.Contains(res.HTML, "inv_8f3a2b") || !strings.Contains(res.Text, "inv_8f3a2b") {
				t.Fatal("code fallback missing")
			}
		})
	}
}

func TestRender_org_invite_blank_accept_url_is_absent(t *testing.T) {
	vars := orgInviteVars("en")
	vars["accept_url"] = "   "
	res, err := Render("org_invite", vars)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.HTML, "Accept the invitation") || strings.Contains(res.Text, "open this link") {
		t.Fatal("blank accept_url rendered the link block")
	}
}

func TestRender_org_invite_refuses_bad_accept_url(t *testing.T) {
	for _, bad := range []string{
		"javascript:alert(1)#inv_8f3a2b",
		"ftp://acme.example.com/invite#inv_8f3a2b",
		"https://#inv_8f3a2b",
		"/invite#inv_8f3a2b",
		"not a url",
		"https://acme.example.com/invite\n#inv_8f3a2b",
	} {
		vars := orgInviteVars("en")
		vars["accept_url"] = bad
		if _, err := Render("org_invite", vars); err == nil {
			t.Fatalf("accept_url %q accepted, want error", bad)
		}
	}
}
