package ziparchive

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
)

var _ ports.Archiver = (*Archiver)(nil)

type Archiver struct{}

func NewArchiver() *Archiver { return &Archiver{} }

func (a *Archiver) Zip(ctx context.Context, srcDir, destZipPath string) error {
	entries, err := filepath.Glob(filepath.Join(srcDir, "*"))
	if err != nil {
		return fmt.Errorf("glob src: %w", err)
	}
	zf, err := os.Create(destZipPath)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	for _, path := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := addFile(zw, path); err != nil {
			return err
		}
	}
	return nil
}

func addFile(zw *zip.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = filepath.Base(path)
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}
