// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.

package telegram

import (
	"errors"
	"net/http"
	"testing"

	"donetick.com/core/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestNoTokenSendsNothing: a server with no Telegram token makes no request to
// Telegram at start and runs without Telegram notifications.
func TestNoTokenSendsNothing(t *testing.T) {
	saved := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected request to %s", r.URL.Host)
		return nil, errors.New("this test allows no requests")
	})
	defer func() { http.DefaultTransport = saved }()

	if notifier := NewTelegramNotifier(&config.Config{}); notifier != nil {
		t.Fatalf("NewTelegramNotifier without a token = %v, want nil", notifier)
	}
}
