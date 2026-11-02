package mailtemplates

import (
	"bytes"
	"embed"
	"fmt"
	htmltmpl "html/template"
	"strings"
	texttmpl "text/template"
)

//go:embed compiled/*
var compiled embed.FS

type Result struct {
	Subject string
	Text    string
	HTML    string
}

const (
	defaultLocale = "fa"
	englishLocale = "en"
)

var subjects = map[string]string{
	"otp_login":        "Your Manova login code",
	"otp_login_sms":    "Login code",
	"invite_developer": "Welcome to Manova — developer workspace invitation",
	"owner_challenge":  "Orbit — server ownership verification",
}

// localizedSubjects holds the subject per locale for templates that ship a
// Persian and an English variant. The matching files are <name>.<locale>.*.
var localizedSubjects = map[string]map[string]string{
	"org_invite": {
		defaultLocale: "دعوت به سازمان — Manova",
		englishLocale: "You have been invited to an organization — Manova",
	},
	"signup_verify": {
		defaultLocale: "کد تأیید Manova",
		englishLocale: "Your Manova verification code",
	},
}

var required = map[string][]string{
	"otp_login":        {"code"},
	"otp_login_sms":    {"code"},
	"invite_developer": {"token", "cli_command", "curl_command"},
	"owner_challenge":  {"code", "email"},
	"org_invite":       {"org_name", "inviter_name", "accept_token", "expires_at"},
	"signup_verify":    {"code", "expires_in_minutes"},
}

var htmlNames = map[string]bool{
	"otp_login":        true,
	"invite_developer": true,
	"owner_challenge":  true,
	"org_invite":       true,
	"signup_verify":    true,
}

var localizedNames = map[string]bool{
	"org_invite":    true,
	"signup_verify": true,
}

func Render(name string, vars map[string]string) (Result, error) {
	if !knownTemplate(name) {
		return Result{}, fmt.Errorf("mailtemplates: unknown template %q", name)
	}
	if vars == nil {
		vars = map[string]string{}
	}
	vars = withDefaults(name, vars)
	locale := vars["locale"]
	subject, err := subjectFor(name, locale)
	if err != nil {
		return Result{}, err
	}
	for _, key := range required[name] {
		if strings.TrimSpace(vars[key]) == "" {
			return Result{}, fmt.Errorf("mailtemplates: %s requires %s", name, key)
		}
	}
	base := name
	if localizedNames[name] {
		base = name + "." + locale
	}
	text, err := execText(base+".txt", vars)
	if err != nil {
		return Result{}, err
	}
	var html string
	if htmlNames[name] {
		html, err = execHTML(base+".html", vars)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{Subject: subject, Text: text, HTML: html}, nil
}

func knownTemplate(name string) bool {
	if _, ok := subjects[name]; ok {
		return true
	}
	_, ok := localizedSubjects[name]
	return ok
}

func subjectFor(name, locale string) (string, error) {
	if subject, ok := subjects[name]; ok {
		return subject, nil
	}
	byLocale, ok := localizedSubjects[name]
	if !ok {
		return "", fmt.Errorf("mailtemplates: unknown template %q", name)
	}
	subject, ok := byLocale[locale]
	if !ok {
		return "", fmt.Errorf("mailtemplates: unknown locale %q for %s", locale, name)
	}
	return subject, nil
}

func withDefaults(name string, vars map[string]string) map[string]string {
	out := make(map[string]string, len(vars)+3)
	for k, v := range vars {
		out[k] = v
	}
	out["locale"] = normalizeLocale(out["locale"])
	if out["expires_in"] == "" {
		out["expires_in"] = "10 minutes"
	}
	if name == "invite_developer" && strings.TrimSpace(out["name"]) == "" {
		out["name"] = "Developer"
	}
	if out["preheader"] == "" {
		switch name {
		case "otp_login":
			out["preheader"] = "Use this code to sign in. It expires in " + out["expires_in"] + "."
		case "invite_developer":
			out["preheader"] = "Your Manova developer workspace invitation is ready."
		case "owner_challenge":
			out["preheader"] = "Confirm server ownership. This code expires in " + out["expires_in"] + "."
		case "org_invite":
			if out["locale"] == englishLocale {
				out["preheader"] = out["inviter_name"] + " invited you to " + out["org_name"] + "."
			} else {
				out["preheader"] = out["inviter_name"] + " شما را به " + out["org_name"] + " دعوت کرده است."
			}
		case "signup_verify":
			if out["locale"] == englishLocale {
				out["preheader"] = "Use this code to verify your new Manova account. It expires in " + out["expires_in_minutes"] + " minutes."
			} else {
				out["preheader"] = "این کد را برای تأیید حساب جدید Manova وارد کنید. اعتبار: " + out["expires_in_minutes"] + " دقیقه."
			}
		}
	}
	return out
}

func normalizeLocale(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), englishLocale) {
		return englishLocale
	}
	return defaultLocale
}

func execText(file string, vars map[string]string) (string, error) {
	raw, err := compiled.ReadFile("compiled/" + file)
	if err != nil {
		return "", err
	}
	t, err := texttmpl.New(file).Parse(string(raw))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func execHTML(file string, vars map[string]string) (string, error) {
	raw, err := compiled.ReadFile("compiled/" + file)
	if err != nil {
		return "", err
	}
	t, err := htmltmpl.New(file).Parse(string(raw))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
