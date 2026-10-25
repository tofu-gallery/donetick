// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream
// Donetick. It is licensed under the AGPL-3.0, like the rest of this server.

package main

import "testing"

func TestPublicOrigin(t *testing.T) {
	cases := map[string]string{
		"https://donetick-ab12.trytofu.app":    "https://donetick-ab12.trytofu.app",
		"https://Donetick.Example.com/":        "https://donetick.example.com",
		"http://localhost:2021":                "http://localhost:2021",
		"chores.example.com":                   "https://chores.example.com",
		"https://chores.example.com/some/path": "https://chores.example.com",
		"":                                     "",
		"   ":                                  "",
	}
	for in, want := range cases {
		if got := publicOrigin(in); got != want {
			t.Errorf("publicOrigin(%q) = %q, want %q", in, got, want)
		}
	}
}
