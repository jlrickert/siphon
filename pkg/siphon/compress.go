package siphon

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

// Compressor provides a symmetric compression/decompression interface.
type Compressor interface {
	Compress(dst io.Writer, src io.Reader) error
	Decompress(dst io.Writer, src io.Reader) error
	Extension() string
}

// NewCompressor returns a Compressor for the given algorithm name.
// Supported names: "zstd", "gzip", "none" (or "").
func NewCompressor(name string) (Compressor, error) {
	switch name {
	case "zstd":
		return &zstdCompressor{}, nil
	case "gzip":
		return &gzipCompressor{}, nil
	case "none", "":
		return &noneCompressor{}, nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", name)
	}
}

// --- zstd ---

type zstdCompressor struct{}

func (c *zstdCompressor) Compress(dst io.Writer, src io.Reader) error {
	enc, err := zstd.NewWriter(dst)
	if err != nil {
		return fmt.Errorf("zstd encoder: %w", err)
	}
	if _, err := io.Copy(enc, src); err != nil {
		enc.Close()
		return fmt.Errorf("zstd compress: %w", err)
	}
	return enc.Close()
}

func (c *zstdCompressor) Decompress(dst io.Writer, src io.Reader) error {
	dec, err := zstd.NewReader(src)
	if err != nil {
		return fmt.Errorf("zstd decoder: %w", err)
	}
	defer dec.Close()
	if _, err := io.Copy(dst, dec); err != nil {
		return fmt.Errorf("zstd decompress: %w", err)
	}
	return nil
}

func (c *zstdCompressor) Extension() string { return ".zst" }

// --- gzip ---

type gzipCompressor struct{}

func (c *gzipCompressor) Compress(dst io.Writer, src io.Reader) error {
	w := gzip.NewWriter(dst)
	if _, err := io.Copy(w, src); err != nil {
		w.Close()
		return fmt.Errorf("gzip compress: %w", err)
	}
	return w.Close()
}

func (c *gzipCompressor) Decompress(dst io.Writer, src io.Reader) error {
	r, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer r.Close()
	if _, err := io.Copy(dst, r); err != nil {
		return fmt.Errorf("gzip decompress: %w", err)
	}
	return nil
}

func (c *gzipCompressor) Extension() string { return ".gz" }

// --- none (passthrough) ---

type noneCompressor struct{}

func (c *noneCompressor) Compress(dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, src)
	return err
}

func (c *noneCompressor) Decompress(dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, src)
	return err
}

func (c *noneCompressor) Extension() string { return "" }
