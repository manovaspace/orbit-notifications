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

var subjects = map[string]string{
	"otp_login":        "Your Manova login code",
	"otp_login_sms":    "Login code",
	"invite_developer": "Welcome to Manova — developer workspace invitation",
	"owner_challenge":  "Orbit — server ownership verification",
}

var required = map[string][]string{
	"otp_login":        {"code"},
	"otp_login_sms":    {"code"},
	"invite_developer": {"token", "cli_command", "curl_command"},
	"owner_challenge":  {"code", "email"},
}

var htmlNames = map[string]bool{
	"otp_login":        true,
	"invite_developer": true,
	"owner_challenge":  true,
}

func Render(name string, vars map[string]string) (Result, error) {
	if _, ok := subjects[name]; !ok {
		return Result{}, fmt.Errorf("mailtemplates: unknown template %q", name)
	}
	if vars == nil {
		vars = map[string]string{}
	}
	vars = withDefaults(name, vars)
	for _, key := range required[name] {
		if strings.TrimSpace(vars[key]) == "" {
			return Result{}, fmt.Errorf("mailtemplates: %s requires %s", name, key)
		}
	}
	text, err := execText(name+".txt", vars)
	if err != nil {
		return Result{}, err
	}
	var html string
	if htmlNames[name] {
		html, err = execHTML(name+".html", vars)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{Subject: subjects[name], Text: text, HTML: html}, nil
}

func withDefaults(name string, vars map[string]string) map[string]string {
	out := make(map[string]string, len(vars)+3)
	for k, v := range vars {
		out[k] = v
	}
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
		}
	}
	return out
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
