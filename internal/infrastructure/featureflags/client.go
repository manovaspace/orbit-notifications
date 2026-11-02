package featureflags

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	unleash "github.com/Unleash/unleash-client-go/v4"
	unleashProvider "github.com/open-feature/go-sdk-contrib/providers/unleash/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

// Evaluator wraps OpenFeature for boolean flag checks.
type Evaluator struct {
	client *openfeature.Client
}

// NewFromEnv connects to Unleash when configured. Provider registration is
// non-blocking: the OpenFeature SDK initialises asynchronously, so evaluations
// return the safe default until the provider is ready (ponytail: no retry loop;
// restart to reconnect).
func NewFromEnv(appName string, log func(msg string, args ...any)) *Evaluator {
	url := envOr("UNLEASH_URL", "http://localhost:10050/api/")
	if isDisabledURL(url) {
		log("feature_flags_disabled", "UNLEASH_URL", url)
		return &Evaluator{}
	}
	token := envOr("UNLEASH_API_TOKEN", "default:development.unleash-insecure-api-token")
	name := envOr("UNLEASH_APP_NAME", appName)

	providerConfig := unleashProvider.ProviderConfig{
		Options: []unleash.ConfigOption{
			unleash.WithUrl(url),
			unleash.WithAppName(name),
			unleash.WithCustomHeaders(http.Header{
				"Authorization": {token},
			}),
			unleash.WithRefreshInterval(15 * time.Second),
		},
	}

	provider, err := unleashProvider.NewProvider(providerConfig)
	if err != nil {
		log("feature_flags_init_failed", "error", err)
		return &Evaluator{}
	}
	if err := openfeature.SetProvider(provider); err != nil {
		log("feature_flags_init_failed", "error", err)
		return &Evaluator{}
	}

	return &Evaluator{client: openfeature.NewClient(name)}
}

// isDisabledURL reports whether UNLEASH_URL was explicitly set to opt out of
// Unleash entirely.
func isDisabledURL(url string) bool {
	url = strings.TrimSpace(url)
	return strings.EqualFold(url, "off") || strings.EqualFold(url, "disabled")
}

// Bool returns the flag value, or defaultVal when the client is unavailable.
func (e *Evaluator) Bool(ctx context.Context, flag string, defaultVal bool) bool {
	if e == nil || e.client == nil {
		return defaultVal
	}
	val, err := e.client.BooleanValue(ctx, flag, defaultVal, openfeature.EvaluationContext{})
	if err != nil {
		return defaultVal
	}
	return val
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
