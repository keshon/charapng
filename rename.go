package charapng

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TokenType is the kind of template segment.
type TokenType int

const (
	TokenLiteral TokenType = iota
	TokenField
)

// Token is a segment of a rename template (literal text or a field placeholder).
type Token struct {
	Type  TokenType
	Value string
}

// ParseTemplate parses a template string into a slice of tokens.
// Placeholders: [name], [creator], [spec], [tag1]..[tagN], [year], [month], [day], [index].
// [spec] is card version (v1/v2/v3); [year]/[month]/[day] use file modification date when path is set.
func ParseTemplate(t string) ([]Token, error) {
	if t == "" {
		return nil, nil
	}
	var out []Token
	re := regexp.MustCompile(`\[([a-zA-Z0-9]+)\]`)
	last := 0
	for _, loc := range re.FindAllStringSubmatchIndex(t, -1) {
		if loc[0] > last {
			out = append(out, Token{Type: TokenLiteral, Value: t[last:loc[0]]})
		}
		out = append(out, Token{Type: TokenField, Value: strings.ToLower(re.ReplaceAllString(t[loc[0]:loc[1]], "$1"))})
		last = loc[1]
	}
	if last < len(t) {
		out = append(out, Token{Type: TokenLiteral, Value: t[last:]})
	}
	return out, nil
}

// RenderFilename renders tokens into a filename for the given card, index, and optional file path.
// path is used for [year], [month], [day] (file modification date); if empty, current date is used.
// index is 1-based for [index]. Missing fields (e.g. no tag2) cause that token to be skipped.
func RenderFilename(tokens []Token, card *Card, index int, path string) string {
	fileTime := time.Now()
	if path != "" {
		if info, err := os.Stat(path); err == nil {
			fileTime = info.ModTime()
		}
	}
	var b strings.Builder
	for _, tok := range tokens {
		if tok.Type == TokenLiteral {
			b.WriteString(tok.Value)
			continue
		}
		val := fieldValue(tok.Value, card, index, fileTime)
		if val == "" {
			continue
		}
		b.WriteString(sanitizeFilename(val))
	}
	s := b.String()
	// Trim only leading separators and trailing spaces so we never strip extension dots
	s = strings.TrimLeft(s, ".- \t")
	s = strings.TrimRight(s, " \t")
	if s == "" {
		return "unnamed"
	}
	return s
}

func fieldValue(key string, card *Card, index int, fileTime time.Time) string {
	switch key {
	case "year":
		return strconv.Itoa(fileTime.Year())
	case "month":
		return fmt.Sprintf("%02d", fileTime.Month())
	case "day":
		return fmt.Sprintf("%02d", fileTime.Day())
	case "index":
		return strconv.Itoa(index)
	}
	if card == nil {
		return ""
	}
	switch key {
	case "name":
		return card.Name
	case "creator":
		return card.Creator
	case "spec":
		return card.SpecVersion
	case "tag1", "tag2", "tag3":
		n := 0
		switch key {
		case "tag1":
			n = 0
		case "tag2":
			n = 1
		case "tag3":
			n = 2
		}
		if n < len(card.Tags) {
			return card.Tags[n]
		}
		return ""
	default:
		// tagN for N > 3
		if strings.HasPrefix(key, "tag") {
			if i, err := strconv.Atoi(strings.TrimPrefix(key, "tag")); err == nil && i >= 1 {
				idx := i - 1
				if idx < len(card.Tags) {
					return card.Tags[idx]
				}
			}
		}
		return ""
	}
}

// invalidFilenameRunes are characters that must not appear in a filename (Windows + Unix).
const invalidFilenameRunes = `/\:*?"<>|`

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(invalidFilenameRunes, r) || r < 32 {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// AvailableFields returns the list of field names that have at least one value
// in the given card set (e.g. for template hints). Always includes name, creator,
// spec, tag1..tagN, year, month, day, index.
func AvailableFields(files []CardFile) []string {
	seen := map[string]struct{}{
		"name": {}, "creator": {}, "spec": {}, "year": {}, "month": {}, "day": {}, "index": {},
	}
	maxTag := 0
	for _, f := range files {
		if f.Card == nil {
			continue
		}
		if len(f.Card.Tags) > maxTag {
			maxTag = len(f.Card.Tags)
		}
	}
	for i := 1; i <= maxTag; i++ {
		seen[fmt.Sprintf("tag%d", i)] = struct{}{}
	}
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	return out
}

// RenamePreview holds old path and proposed new name (base only) for one file.
type RenamePreview struct {
	Path    string
	NewName string
}

// BatchRenamePreview generates new filenames for the given files using the template.
// The template should include the extension (e.g. [name]-[tag1].png). Returns a list of (path, newName).
func BatchRenamePreview(files []CardFile, template string) ([]RenamePreview, error) {
	tokens, err := ParseTemplate(template)
	if err != nil {
		return nil, err
	}
	out := make([]RenamePreview, 0, len(files))
	for i, f := range files {
		newName := RenderFilename(tokens, f.Card, i+1, f.Path)
		if newName == "unnamed" || strings.TrimSpace(newName) == "" {
			newName = filepath.Base(f.Path)
		}
		// Always preserve source file extension so we never lose .png
		if ext := filepath.Ext(f.Path); ext != "" {
			currentExt := filepath.Ext(newName)
			if currentExt != ext {
				newName = strings.TrimSuffix(newName, currentExt) + ext
			}
		}
		// Safety: ensure result is a single basename (no path components)
		if base := filepath.Base(newName); base != "" {
			newName = base
		}
		out = append(out, RenamePreview{Path: f.Path, NewName: newName})
	}
	return out, nil
}

var ErrCollision = errors.New("charapng: rename would cause collision or overwrite")

// BatchRename renames files according to the template. It checks for collisions
// (two files mapping to the same name, or target already exists and is different file)
// and returns ErrCollision without renaming any file. Otherwise renames and returns nil.
func BatchRename(files []CardFile, template string) error {
	if len(files) == 0 {
		return nil
	}
	previews, err := BatchRenamePreview(files, template)
	if err != nil {
		return err
	}
	// Collision: same directory + same NewName from different source paths
	byDir := make(map[string]map[string]string)
	for _, p := range previews {
		dir := filepath.Dir(p.Path)
		if byDir[dir] == nil {
			byDir[dir] = make(map[string]string)
		}
		if existing, ok := byDir[dir][p.NewName]; ok && existing != p.Path {
			return ErrCollision
		}
		byDir[dir][p.NewName] = p.Path
		full := filepath.Join(dir, p.NewName)
		if full != p.Path {
			if _, err := os.Stat(full); err == nil {
				return ErrCollision
			}
		}
	}
	for _, p := range previews {
		dir := filepath.Dir(p.Path)
		full := filepath.Join(dir, p.NewName)
		if p.Path == full {
			continue
		}
		if err := os.Rename(p.Path, full); err != nil {
			return fmt.Errorf("charapng: rename %q -> %q: %w", p.Path, full, err)
		}
	}
	return nil
}
