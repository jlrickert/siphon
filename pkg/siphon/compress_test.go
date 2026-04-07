package siphon_test

import (
	"bytes"
	"testing"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func TestCompressor_GzipRoundTrip(t *testing.T) {
	t.Parallel()

	comp, err := siphon.NewCompressor("gzip")
	require.NoError(t, err)

	original := []byte("Hello, this is test data for gzip compression round-trip testing.")

	var compressed bytes.Buffer
	err = comp.Compress(&compressed, bytes.NewReader(original))
	require.NoError(t, err)
	require.NotEmpty(t, compressed.Bytes())
	// Compressed data should be different from original (for non-trivial input).
	require.NotEqual(t, original, compressed.Bytes())

	var decompressed bytes.Buffer
	err = comp.Decompress(&decompressed, bytes.NewReader(compressed.Bytes()))
	require.NoError(t, err)
	require.Equal(t, original, decompressed.Bytes())
}

func TestCompressor_ZstdRoundTrip(t *testing.T) {
	t.Parallel()

	comp, err := siphon.NewCompressor("zstd")
	require.NoError(t, err)

	original := []byte("Hello, this is test data for zstd compression round-trip testing.")

	var compressed bytes.Buffer
	err = comp.Compress(&compressed, bytes.NewReader(original))
	require.NoError(t, err)
	require.NotEmpty(t, compressed.Bytes())

	var decompressed bytes.Buffer
	err = comp.Decompress(&decompressed, bytes.NewReader(compressed.Bytes()))
	require.NoError(t, err)
	require.Equal(t, original, decompressed.Bytes())
}

func TestCompressor_NonePassthrough(t *testing.T) {
	t.Parallel()

	comp, err := siphon.NewCompressor("none")
	require.NoError(t, err)

	original := []byte("passthrough data")

	var compressed bytes.Buffer
	err = comp.Compress(&compressed, bytes.NewReader(original))
	require.NoError(t, err)
	require.Equal(t, original, compressed.Bytes())

	var decompressed bytes.Buffer
	err = comp.Decompress(&decompressed, bytes.NewReader(compressed.Bytes()))
	require.NoError(t, err)
	require.Equal(t, original, decompressed.Bytes())
}

func TestCompressor_EmptyStringIsNone(t *testing.T) {
	t.Parallel()

	comp, err := siphon.NewCompressor("")
	require.NoError(t, err)
	require.Equal(t, "", comp.Extension())
}

func TestCompressor_Unsupported(t *testing.T) {
	t.Parallel()

	_, err := siphon.NewCompressor("lz4")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported")
}

func TestCompressor_Extensions(t *testing.T) {
	t.Parallel()

	gzip, _ := siphon.NewCompressor("gzip")
	require.Equal(t, ".gz", gzip.Extension())

	zstd, _ := siphon.NewCompressor("zstd")
	require.Equal(t, ".zst", zstd.Extension())

	none, _ := siphon.NewCompressor("none")
	require.Equal(t, "", none.Extension())
}
