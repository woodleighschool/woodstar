package web

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var (
	// Dist embeds the built Vite bundle.
	//go:embed all:dist
	Dist embed.FS

	// DistDirFS exposes the built Vite bundle rooted at dist.
	DistDirFS = MustSubFS(Dist, "dist")
)

type defaultFS struct {
	prefix string
	fs     fs.FS
}

func (f defaultFS) Open(name string) (fs.File, error) {
	if f.fs == nil {
		return os.Open(name)
	}
	return f.fs.Open(name)
}

// MustSubFS returns fsRoot as a sub-filesystem or panics.
func MustSubFS(currentFS fs.FS, fsRoot string) fs.FS {
	subFS, err := subFS(currentFS, fsRoot)
	if err != nil {
		panic(fmt.Errorf("create sub fs: %w", err))
	}
	return subFS
}

func subFS(currentFS fs.FS, root string) (fs.FS, error) {
	root = filepath.ToSlash(filepath.Clean(root))
	if dFS, ok := currentFS.(*defaultFS); ok {
		if !filepath.IsAbs(root) {
			root = filepath.Join(dFS.prefix, root)
		}
		return &defaultFS{
			prefix: root,
			fs:     os.DirFS(root),
		}, nil
	}
	return fs.Sub(currentFS, root)
}
