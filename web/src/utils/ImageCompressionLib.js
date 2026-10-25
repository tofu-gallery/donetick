// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.
//
// browser-image-compression shrinks an image in a web worker, which loads the library's
// own script from cdn.jsdelivr.net unless it is given another address (its libURL
// option). This is that script as the app's own file (Vite's ?url import copies it into
// the build), so shrinking an image before an upload asks no other host for anything.
// The worker starts from a blob: address, so the script's address has to be absolute.
import imageCompressionScript from 'browser-image-compression/dist/browser-image-compression.js?url'

export const imageCompressionLibURL = new URL(
  imageCompressionScript,
  window.location.href,
).href
