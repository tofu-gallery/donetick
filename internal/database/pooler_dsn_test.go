// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream
// Donetick. It is licensed under the AGPL-3.0, like the rest of this server.

package database

import "testing"

func TestPoolerSafeDSN(t *testing.T) {
	cases := map[string]string{
		"postgresql://u:p@db.example:6543/postgres?sslmode=require":    "postgresql://u:p@db.example:6543/postgres?sslmode=require&default_query_exec_mode=cache_describe",
		"postgres://u:p@db.example:5432/postgres":                      "postgres://u:p@db.example:5432/postgres?default_query_exec_mode=cache_describe",
		"postgresql://u:p@h/d?default_query_exec_mode=simple_protocol": "postgresql://u:p@h/d?default_query_exec_mode=simple_protocol",
		"host=h port=5432 user=u dbname=d sslmode=require":             "host=h port=5432 user=u dbname=d sslmode=require default_query_exec_mode=cache_describe",
	}
	for in, want := range cases {
		if got := poolerSafeDSN(in); got != want {
			t.Errorf("poolerSafeDSN(%q) = %q, want %q", in, got, want)
		}
	}
}
