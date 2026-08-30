package templates

import "github.com/manovaspace/orbit-notifications/pkg/mailtemplates"

// Render applies template vars to a named catalog template.
func Render(name string, vars map[string]string) (mailtemplates.Result, error) {
	return mailtemplates.Render(name, vars)
}
