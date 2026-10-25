#!/bin/sh
# Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
# It is licensed under the AGPL-3.0, like the rest of this repository.
#
# Maps the variables Tofu sets on every app onto Donetick's own DT_* settings, then
# runs the image's command (/donetick):
#   PORT          -> DT_SERVER_PORT. Tofu sends traffic to PORT, so PORT wins over
#                    DT_SERVER_PORT; without PORT the configured port (2021) stays.
#   DATABASE_URL  -> DT_DATABASE_TYPE=postgres, unless DT_DATABASE_TYPE is set. The
#                    server connects with DATABASE_URL itself while DT_DATABASE_HOST
#                    is unset (internal/database/database.go).
#   DT_JWT_SECRET -> derived from DATABASE_URL when it is not set (below).
set -eu

if [ -n "${PORT:-}" ]; then
	DT_SERVER_PORT=$PORT
	export DT_SERVER_PORT
fi

if [ -n "${DATABASE_URL:-}" ]; then
	: "${DT_DATABASE_TYPE:=postgres}"
	export DT_DATABASE_TYPE
fi

# Donetick signs its sign-in tokens with DT_JWT_SECRET and refuses to start without a
# strong one, and the settings file in this image holds upstream's placeholder, which it
# refuses. Tofu does not supply a secret, so it is derived here from the database URL, a
# hidden value only the server sees: a copy starts with nothing set by hand and keeps the
# same secret across restarts. Anything that changes DATABASE_URL changes the secret at
# the next start (see TOFU.md). A secret the owner sets always wins. Without DATABASE_URL
# there is nothing to derive it from, so a random one is made for this start only, as
# upstream does on a filesystem that keeps nothing: everyone signs in again after a restart.
if [ -z "${DT_JWT_SECRET:-}" ]; then
	if [ -n "${DATABASE_URL:-}" ]; then
		DT_JWT_SECRET=$(printf '%s' "donetick-jwt-secret:${DATABASE_URL}" | sha256sum | cut -d' ' -f1)
	else
		echo "tofu-entrypoint: neither DT_JWT_SECRET nor DATABASE_URL is set, so this start uses a random sign-in secret" >&2
		DT_JWT_SECRET=$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')
	fi
	export DT_JWT_SECRET
fi

exec "$@"
