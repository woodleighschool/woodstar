package destination

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/woodleighschool/stemma/plugin"
)

func readPkginfo(ctx context.Context, artifact plugin.Artifact) (json.RawMessage, error) {
	if artifact.Size > 8<<20 {
		return nil, errors.New("pkginfo exceeds 8 MiB")
	}
	var data bytes.Buffer
	if err := verifyArtifact(ctx, artifact, &data); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
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
