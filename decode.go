package charapng

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrCardNotFound is returned when no recognised character card payload is found in the PNG.
var ErrCardNotFound = errors.New("charapng: character card not found")

// keywords is the set of tEXt chunk keys that may carry character card data.
var keywords = map[string]struct{}{
	"chara":     {},
	"character": {},
	"card":      {},
	"ccv3":      {},
}

// DecodeFile opens the file at path and extracts the character card embedded inside it.
// It is a convenience wrapper around Decode.
func DecodeFile(path string) (*Card, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("charapng: open file: %w", err)
	}
	defer f.Close()

	return Decode(f)
}

// Decode reads a PNG from r and returns the first character card payload it finds.
// Returns ErrCardNotFound if no recognised payload exists.
func Decode(r io.Reader) (*Card, error) {
	chunks, err := readPNGChunks(r)
	if err != nil {
		return nil, fmt.Errorf("charapng: read png: %w", err)
	}

	texts, err := extractTextChunks(chunks)
	if err != nil {
		return nil, fmt.Errorf("charapng: extract text chunks: %w", err)
	}

	for _, t := range texts {
		if !isKeyword(t.Keyword) {
			continue
		}

		data, err := base64.StdEncoding.DecodeString(string(t.Value))
		if err != nil {
			return nil, fmt.Errorf("charapng: decode base64 for keyword %q: %w", t.Keyword, err)
		}

		return &Card{
			Keyword: t.Keyword,
			JSON:    data,
		}, nil
	}

	return nil, ErrCardNotFound
}

func isKeyword(k string) bool {
	_, ok := keywords[k]
	return ok
}
