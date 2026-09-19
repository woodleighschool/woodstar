package destination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// validateInstaller checks the installer against the pkginfo describing it.
func validateInstaller(ctx context.Context, installer plugin.Artifact, installerType packages.InstallerType, installerItemHash string) error {
	if installerType == packages.InstallerTypeNoPkg {
		if installer.Path != "" || installer.SHA256 != "" {
			return errors.New("nopkg must not include installer content")
		}
		return nil
	}
	if installer.Path == "" {
		return errors.New("installer is required")
	}
	if installerItemHash != "" && installerItemHash != installer.SHA256 {
		return errors.New("pkginfo installer_item_hash does not match installer")
	}
	if err := verifyArtifact(ctx, installer, nil); err != nil {
		return fmt.Errorf("installer: %w", err)
	}
	return nil
}

func validateIcon(ctx context.Context, artifact plugin.Artifact) error {
	if artifact.Format != "png" || artifact.Size <= 0 || artifact.Size > 32<<20 {
		return errors.New("icon input requires a PNG artifact no larger than 32 MiB")
	}
	return verifyArtifact(ctx, artifact, nil)
}

func validateArtifact(artifact plugin.Artifact) error {
	digest, err := hex.DecodeString(artifact.SHA256)
	if err != nil || len(digest) != 32 || strings.ToLower(artifact.SHA256) != artifact.SHA256 {
		return errors.New("installer artifact requires a lowercase SHA-256 digest")
	}
	if artifact.Filename == "" || strings.TrimSpace(artifact.Filename) != artifact.Filename || filepath.Base(artifact.Filename) != artifact.Filename || artifact.Size < 0 {
		return errors.New("installer artifact requires an unpadded base filename and nonnegative size")
	}
	return nil
}

func verifyArtifact(ctx context.Context, artifact plugin.Artifact, output io.Writer) error {
	if err := validateArtifact(artifact); err != nil {
		return err
	}
	if artifact.Tree || !filepath.IsAbs(artifact.Path) {
		return errors.New("artifact must be an absolute leased file")
	}
	file, err := os.Open(artifact.Path)
	if err != nil {
		return errors.New("open leased artifact")
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != artifact.Size {
		return errors.New("leased artifact size or file type changed")
	}
	hash := sha256.New()
	var writer io.Writer = hash
	if output != nil {
		writer = io.MultiWriter(hash, output)
	}
	n, err := io.Copy(writer, io.LimitReader(artifactReader{Reader: file, ctx: ctx}, artifact.Size+1))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("read leased artifact")
	}
	if n != artifact.Size || hex.EncodeToString(hash.Sum(nil)) != artifact.SHA256 {
		return errors.New("leased artifact digest changed")
	}
	return ctx.Err()
}

type artifactReader struct {
	io.Reader

	ctx context.Context
}

func (r artifactReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}
