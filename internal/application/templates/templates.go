package templates

import (
	"fmt"
	"os"
	"strings"
)

// Render applies template vars to a named template.
func Render(name string, vars map[string]string) (subject, body string, err error) {
	switch name {
	case "otp_login":
		code := vars["code"]
		if code == "" {
			return "", "", fmt.Errorf("templates: otp_login requires code var")
		}
		return "Your login code", fmt.Sprintf("Your one-time login code is: %s\n\nIt expires in 10 minutes.", code), nil
	case "otp_login_sms":
		code := vars["code"]
		if code == "" {
			return "", "", fmt.Errorf("templates: otp_login_sms requires code var")
		}
		return "Login code", fmt.Sprintf("Your one-time login code is: %s\n\nIt expires in 10 minutes.", code), nil
	default:
		if os.Getenv("DEPLOYMENT_ENVIRONMENT") != "dev" {
			return "", "", fmt.Errorf("templates: unknown template %q", name)
		}
		return name, strings.Join(mapPairs(vars), "\n"), nil
	}
}

func mapPairs(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}
