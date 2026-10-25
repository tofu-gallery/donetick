// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.
//
// A copy keeps uploaded files in a storage bucket its owner adds with the DT_STORAGE_*
// settings (TOFU.md). Until then both upload routes answer 503 with code
// "uploads_not_set_up" and one plain sentence saying so
// (internal/storage/uploads_set_up.go), and every place that uploads a file shows that
// sentence instead of a generic or misleading error.

const UPLOADS_NOT_SET_UP_CODE = 'uploads_not_set_up'

export const UPLOADS_NOT_SET_UP_TITLE = 'Uploads are not set up'

// The server's sentence, for an answer that carries the code without it.
const UPLOADS_NOT_SET_UP_MESSAGE =
  "Uploading files is not set up yet: the owner of this app has to add a storage bucket with the DT_STORAGE_* settings (see TOFU.md in the app's source code)."

// Returns the { title, message } to show when an upload's response says uploads are not
// set up, and null for any other response (a 503 from elsewhere included).
export const uploadsNotSetUpError = async response => {
  if (!response || response.status !== 503) return null
  try {
    const body = await response.clone().json()
    if (body?.code !== UPLOADS_NOT_SET_UP_CODE) return null
    return {
      title: UPLOADS_NOT_SET_UP_TITLE,
      message: body.error || UPLOADS_NOT_SET_UP_MESSAGE,
    }
  } catch {
    return null
  }
}
