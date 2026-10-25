// Tofu (2026-10-25): this file is Tofu's addition; it is not part of upstream Donetick.
// It is licensed under the AGPL-3.0, like the rest of this repository.

package storage

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
)

// UploadsNotSetUpCode marks the answer below, so the web app can tell it from any
// other 503 (web/src/utils/UploadsNotSetUp.js).
const UploadsNotSetUpCode = "uploads_not_set_up"

// UploadsNotSetUpMessage is the one sentence both upload routes answer with while
// uploaded files have nowhere to go, and the web app shows it as it is.
const UploadsNotSetUpMessage = "Uploading files is not set up yet: the owner of this app has to add a storage bucket with the DT_STORAGE_* settings (see TOFU.md in the app's source code)."

// UploadsSetUp reports whether s has somewhere to keep uploaded files. Local storage
// (storage.storage_type: local) keeps them on the server's own disk. S3 storage, the
// default, needs a bucket and its region: without a bucket name an upload ends in a
// misleading error, and without a region in the S3 client's MissingRegion. public asks
// about the bucket profile photos go to (storage.public_bucket_name, falling back to
// storage.bucket_name); otherwise it is the bucket attachments and images go to.
func UploadsSetUp(s Storage, public bool) bool {
	s3Storage, ok := s.(*S3Storage)
	if !ok {
		return true
	}
	bucket := s3Storage.Bucket
	if public {
		bucket = s3Storage.publicBucket()
	}
	return bucket != "" && aws.StringValue(s3Storage.Client.Config.Region) != ""
}

// RefuseUploadsNotSetUp answers an upload with 503 and UploadsNotSetUpMessage.
func RefuseUploadsNotSetUp(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": UploadsNotSetUpMessage,
		"code":  UploadsNotSetUpCode,
	})
}
