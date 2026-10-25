// Isolates the posthog-js dependency so index.js never touches the SDK
// directly — keeps PostHog-specific config in one place and makes it
// possible to swap/mock the backend later without touching call sites.
//
// Tofu (2026-10-25): modified from upstream. A Tofu copy ships no analytics:
// this module no longer reads VITE_POSTHOG_KEY or imports posthog-js, so the
// build leaves the SDK out and no event is ever sent. isConfigured() is always
// false and getClient() always answers null, which every caller in index.js
// already treats as "analytics off"; the privacy switches in Settings stay and
// have nothing to switch.

let posthog = null

export const isConfigured = () => false

export const getClient = async () => null

export const getClientSync = () => posthog
