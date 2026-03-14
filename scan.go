package charapng

import (
	"os"
	"path/filepath"
	"strings"
)

// CardFile pairs a file path with its extracted normalized card metadata.
// Card may be nil if metadata was not extracted (e.g. not a character card);
// ScanDirectory only returns entries where Card is non-nil.
type CardFile struct {
	Path string
	Card *Card
}

// ScanDirectory walks dir for *.png files, extracts metadata for each,
// and returns only successfully parsed character cards. Broken or non-card
// PNGs are skipped; the function does not return an error for those.
func ScanDirectory(dir string) ([]CardFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []CardFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".png") {
			continue
		}
		path := filepath.Join(dir, name)
		card, err := ExtractCardMetadata(path)
		if err != nil {
			continue
		}
		out = append(out, CardFile{Path: path, Card: card})
	}
	return out, nil
}
