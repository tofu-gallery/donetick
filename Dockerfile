# Tofu (2026-10-25): modified from upstream for Tofu's gallery copy. Each change is
# marked "Tofu (2026-10-25)" where it is made, and TOFU.md says why.

# Tofu (2026-10-25): Stage 0 builds the web app in web/ (donetick/frontend, see TOFU.md),
# which upstream's release workflow builds from its own repository before building this
# image. Its install scripts are not run: they download the PostHog CLI and the libvips
# image library, neither of which the web build uses, and print funding notices. The
# app's own postinstall step, patch-package, runs on its own.
FROM node:22-bookworm-slim AS web

WORKDIR /usr/src/web

COPY web/package.json web/package-lock.json ./
COPY web/patches ./patches
RUN npm ci --ignore-scripts --no-audit --no-fund && npx --no-install patch-package

COPY web/ ./
# The two settings web/.env.selfhosted holds for this build, set here so the build never
# depends on that file: the API is this same server, and the app runs as a self-hosted copy.
ENV VITE_APP_API_URL="" VITE_IS_SELF_HOSTED=true
RUN npx --no-install vite build --mode selfhosted

# Stage 1: Build the application from source.
# --platform=$BUILDPLATFORM pins this stage to the host (native) architecture
# even when building for other target platforms, so `go build` cross-compiles
# natively via GOOS/GOARCH instead of running under QEMU emulation. Emulated
# `go build` for arm64/armv7 is drastically slower than native cross-compiling
# a pure-Go (CGO_ENABLED=0) binary -- this matters for the multi-arch
# (linux/amd64,arm64,arm/v7) image built in go-release.yml.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

WORKDIR /usr/src/app

RUN apk --no-cache add ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Tofu (2026-10-25): the server embeds frontend/dist (go:embed), which only holds a
# placeholder page in this repository; put the web app built above there, as upstream's
# release workflow does.
RUN rm -rf ./frontend/dist
COPY --from=web /usr/src/web/dist ./frontend/dist

# TARGETOS/TARGETARCH are populated automatically by buildkit for the actual
# target platform being built, regardless of the native host running this stage.
ARG TARGETOS
ARG TARGETARCH
# VERSION/COMMIT are baked into config.Version/config.Commit via ldflags so
# `/api/v1/resource` and `/health` report the exact build that's running.
# Passing neither is fine for local/dev builds (falls back to "dev").
ARG VERSION=dev
ARG COMMIT=dev
RUN BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w \
      -X donetick.com/core/config.Version=${VERSION} \
      -X donetick.com/core/config.Commit=${COMMIT} \
      -X donetick.com/core/config.BuildDate=${BUILD_DATE}" \
      -buildvcs=false -o /donetick .

# Stage 2: Create a smaller runtime image
FROM alpine:latest

# Install necessary CA certificates and timezone data
RUN apk --no-cache add ca-certificates libc6-compat tzdata

# Copy the binary and config folder from the builder stage
COPY --from=builder /donetick /donetick
COPY --from=builder /usr/src/app/config /config
# Tofu (2026-10-25): the self-hosted settings file is part of the image, instead of being
# written by the server at its first start, which needs a writable /config and, where
# nothing is kept between starts, writes a new random sign-in secret every time. The
# sign-in secret in it is upstream's placeholder, which the server refuses to start with:
# tofu-entrypoint.sh always sets DT_JWT_SECRET, and every DT_* setting wins over the file.
COPY --from=builder /usr/src/app/config/selfhosted.yaml.example /config/selfhosted.yaml
# Tofu (2026-10-25): maps Tofu's PORT and DATABASE_URL onto DT_* settings, then starts /donetick
COPY tofu-entrypoint.sh /tofu-entrypoint.sh

# Set environment variables; override at `docker run`/compose time to pick
# the config file under /config (e.g. DT_ENV=selfhosted or DT_ENV=prod)
ENV DT_ENV="selfhosted"
# Tofu (2026-10-25): the web app's "Source code" links open the exact source this image is
# built from (AGPL-3.0 section 13); an owner who changes the code points it at theirs.
ENV DT_SOURCE_URL=https://github.com/tofu-gallery/donetick/tree/pin/2026-10-26.1

# Expose the application port
EXPOSE 2021

# Tofu (2026-10-25): checks the port the server listens on (PORT, then DT_SERVER_PORT, then 2021)
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget --quiet --tries=1 --spider "http://localhost:${PORT:-${DT_SERVER_PORT:-2021}}/api/v1/health" || exit 1

# Command to run the application
# Tofu (2026-10-25): starts through tofu-entrypoint.sh, which ends by running the command below
ENTRYPOINT ["/bin/sh", "/tofu-entrypoint.sh"]
CMD ["/donetick"]
