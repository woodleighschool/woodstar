package artifacts

import (
	"fmt"
	"mime"
	"path"
	"strings"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// UploadTargetInput describes the uploaded object Woodstar should reserve.
type UploadTargetInput struct {
	Kind        ArtifactKind
	Filename    string
	DisplayName string
	ContentType string
	SizeBytes   int64
	SHA256      string
}

// BuildUploadTarget returns the artifact mutation for a direct object upload.
func BuildUploadTarget(input UploadTargetInput) (ArtifactMutation, error) {
	filename := cleanUploadFilename(input.Filename)
	if filename == "" {
		return ArtifactMutation{}, fmt.Errorf("%w: filename is required", dbutil.ErrInvalidInput)
	}
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = uploadContentType(filename)
	}
	target := ArtifactMutation{
		Kind:        input.Kind,
		DisplayName: input.DisplayName,
		Location:    uploadLocation(input.SHA256, filename),
		ContentType: contentType,
		SizeBytes:   input.SizeBytes,
		SHA256:      strings.TrimSpace(input.SHA256),
	}
	target.StorageKey = storageKey(target.Kind, target.Location)
	if target.DisplayName == "" {
		target.DisplayName = filename
	}
	if err := target.Validate(); err != nil {
		return ArtifactMutation{}, err
	}
	return target, nil
}

func cleanUploadFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, `\`, "/"))
	filename = path.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		return ""
	}
	return filename
}

func uploadLocation(sha256 string, filename string) string {
	sha256 = strings.TrimSpace(sha256)
	if len(sha256) >= 12 {
		return sha256[:12] + "/" + filename
	}
	return filename
}

func storageKey(kind ArtifactKind, location string) string {
	switch kind {
	case ArtifactKindPackage:
		return "pkgs/" + location
	case ArtifactKindIcon:
		return "icons/" + location
	default:
		return string(kind) + "/" + location
	}
}

func uploadContentType(filename string) string {
	if contentType := mime.TypeByExtension(path.Ext(filename)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}
