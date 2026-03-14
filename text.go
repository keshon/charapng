package charapng

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
)

type textChunk struct {
	Keyword string
	Value   []byte
}

// extractTextChunks collects all tEXt, zTXt, and iTXt chunks from a PNG chunk list.
// Returns an empty (non-nil) slice when none are present — not an error.
func extractTextChunks(chunks []chunk) ([]textChunk, error) {
	var out []textChunk

	for _, c := range chunks {
		var (
			t   textChunk
			err error
		)

		switch c.Type {
		case "tEXt":
			t, err = parseTEXt(c.Data)
		case "zTXt":
			t, err = parseZTXt(c.Data)
		case "iTXt":
			t, err = parseITXt(c.Data)
		default:
			continue
		}

		if err != nil {
			// A single malformed chunk is not fatal; skip it.
			continue
		}

		out = append(out, t)
	}

	return out, nil
}

// parseTEXt parses a raw tEXt chunk: keyword\0value
func parseTEXt(data []byte) (textChunk, error) {
	keyword, value, ok := splitNull(data)
	if !ok {
		return textChunk{}, errors.New("tEXt: missing null separator")
	}
	if len(keyword) == 0 {
		return textChunk{}, errors.New("tEXt: empty keyword")
	}

	return textChunk{Keyword: keyword, Value: value}, nil
}

// parseZTXt parses a raw zTXt chunk: keyword\0<compression method byte><compressed data>
func parseZTXt(data []byte) (textChunk, error) {
	keyword, rest, ok := splitNull(data)
	if !ok {
		return textChunk{}, errors.New("zTXt: missing null separator")
	}
	if len(keyword) == 0 {
		return textChunk{}, errors.New("zTXt: empty keyword")
	}
	// rest[0] is the compression method byte (must be 0 = deflate); skip it.
	if len(rest) < 1 {
		return textChunk{}, errors.New("zTXt: missing compression method byte")
	}

	buf, err := decompressChunk(rest[1:])
	if err != nil {
		return textChunk{}, fmt.Errorf("zTXt: decompress: %w", err)
	}

	return textChunk{Keyword: keyword, Value: buf}, nil
}

// parseITXt parses a raw iTXt chunk.
//
// Layout:
//
//	keyword \0
//	compression-flag (1 byte)
//	compression-method (1 byte)
//	language-tag \0
//	translated-keyword \0
//	text (possibly compressed)
func parseITXt(data []byte) (textChunk, error) {
	keyword, rest, ok := splitNull(data)
	if !ok {
		return textChunk{}, errors.New("iTXt: missing null separator after keyword")
	}
	if len(keyword) == 0 {
		return textChunk{}, errors.New("iTXt: empty keyword")
	}
	if len(rest) < 2 {
		return textChunk{}, errors.New("iTXt: data too short for flags")
	}

	compressed := rest[0] == 1
	// rest[1] is the compression method byte; unused for now.
	rest = rest[2:]

	// Skip language tag.
	_, rest, ok = splitNull(rest)
	if !ok {
		return textChunk{}, errors.New("iTXt: missing null separator after language tag")
	}

	// Skip translated keyword.
	_, rest, ok = splitNull(rest)
	if !ok {
		return textChunk{}, errors.New("iTXt: missing null separator after translated keyword")
	}

	if compressed {
		buf, err := decompressChunk(rest)
		if err != nil {
			return textChunk{}, fmt.Errorf("iTXt: decompress: %w", err)
		}
		return textChunk{Keyword: keyword, Value: buf}, nil
	}

	return textChunk{Keyword: keyword, Value: rest}, nil
}

// splitNull splits b at the first null byte, returning (before, after, true).
// Returns ("", nil, false) when no null byte is found.
func splitNull(b []byte) (keyword string, rest []byte, ok bool) {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return "", nil, false
	}
	return string(b[:i]), b[i+1:], true
}

// decompressChunk tries zlib first (PNG standard), then gzip (used by some generators).
func decompressChunk(data []byte) ([]byte, error) {
	buf, err := zlibDecompress(data)
	if err == nil {
		return buf, nil
	}
	return gzipDecompress(data)
}

func zlibDecompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

func gzipDecompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
