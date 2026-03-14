package charapng

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	tokens, err := ParseTemplate("[name]-[tag1]-[year].png")
	if err != nil {
		t.Fatal(err)
	}
	// [name], -, [tag1], -, [year], .png = 6 tokens
	if len(tokens) != 6 {
		t.Fatalf("got %d tokens, want 6", len(tokens))
	}
	wantTypes := []TokenType{TokenField, TokenLiteral, TokenField, TokenLiteral, TokenField, TokenLiteral}
	for i, tok := range tokens {
		if i >= len(wantTypes) {
			break
		}
		if tok.Type != wantTypes[i] {
			t.Errorf("token %d: type = %v, want %v", i, tok.Type, wantTypes[i])
		}
	}
	if tokens[0].Value != "name" {
		t.Errorf("first token value = %q", tokens[0].Value)
	}
}

func TestRenderFilename(t *testing.T) {
	card := &Card{
		Name:   "Alice",
		Tags:   []string{"fantasy", "mage"},
		Creator: "Bob",
	}
	tokens, _ := ParseTemplate("[name]-[tag1]-[creator].png")
	got := RenderFilename(tokens, card, 1, "")
	if got != "Alice-fantasy-Bob.png" {
		t.Errorf("RenderFilename = %q, want Alice-fantasy-Bob.png", got)
	}
	// card has 2 tags so [tag2] = mage
	tokens2, _ := ParseTemplate("[name]-[tag2].png")
	got2 := RenderFilename(tokens2, card, 1, "")
	if got2 != "Alice-mage.png" {
		t.Errorf("RenderFilename [name]-[tag2].png = %q, want Alice-mage.png", got2)
	}
	// missing tag3: field token is skipped (empty), literals kept
	cardOneTag := &Card{Name: "X", Tags: []string{"only"}}
	tokens3, _ := ParseTemplate("[name]-[tag3].png")
	got3 := RenderFilename(tokens3, cardOneTag, 1, "")
	if got3 != "X-.png" {
		t.Errorf("RenderFilename (missing tag3) = %q, want X-.png", got3)
	}
	// [spec] = card spec version (v1/v2/v3)
	cardWithSpec := &Card{Name: "Alice", SpecVersion: "v2"}
	tokensSpec, _ := ParseTemplate("[name]-[spec].png")
	gotSpec := RenderFilename(tokensSpec, cardWithSpec, 1, "")
	if gotSpec != "Alice-v2.png" {
		t.Errorf("RenderFilename [name]-[spec].png = %q, want Alice-v2.png", gotSpec)
	}
}

func TestBatchRenamePreview(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.png")
	f2 := filepath.Join(dir, "b.png")
	// Create minimal PNGs so CardFile has valid Path
	writeMinimalPNG(f1, "chara", "e30=") // base64 of {}
	writeMinimalPNG(f2, "chara", "e30=")
	card1, _ := ExtractCardMetadata(f1)
	card2, _ := ExtractCardMetadata(f2)
	card1.Name = "First"
	card2.Name = "Second"
	files := []CardFile{
		{Path: f1, Card: card1},
		{Path: f2, Card: card2},
	}
	previews, err := BatchRenamePreview(files, "[name].png")
	if err != nil {
		t.Fatal(err)
	}
	if len(previews) != 2 {
		t.Fatalf("previews length = %d", len(previews))
	}
	if filepath.Base(previews[0].NewName) != "First.png" {
		t.Errorf("preview[0] = %q", previews[0].NewName)
	}
	if filepath.Base(previews[1].NewName) != "Second.png" {
		t.Errorf("preview[1] = %q", previews[1].NewName)
	}
}

func TestBatchRenamePreview_preservesExtension(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "old.png")
	writeMinimalPNG(f1, "chara", "e30=")
	card, _ := ExtractCardMetadata(f1)
	card.Name = "Char"
	files := []CardFile{{Path: f1, Card: card}}

	// Template without .png: must get extension from source
	previews, err := BatchRenamePreview(files, "[name]-[index]")
	if err != nil {
		t.Fatal(err)
	}
	if len(previews) != 1 {
		t.Fatalf("len(previews) = %d", len(previews))
	}
	if want := "Char-1.png"; filepath.Base(previews[0].NewName) != want {
		t.Errorf("NewName = %q, want %q", filepath.Base(previews[0].NewName), want)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"normal", "normal"},
		{"path/to/file", "pathtofile"},
		{"a*b?c", "abc"},
		{"  trim  ", "trim"},
		{"v2.0", "v2.0"},
	}
	for _, tt := range tests {
		got := sanitizeFilename(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBatchRename(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "old1.png")
	f2 := filepath.Join(dir, "old2.png")
	writeMinimalPNG(f1, "chara", "e30=")
	writeMinimalPNG(f2, "chara", "e30=")
	card1, _ := ExtractCardMetadata(f1)
	card2, _ := ExtractCardMetadata(f2)
	card1.Name = "A"
	card2.Name = "B"
	files := []CardFile{
		{Path: f1, Card: card1},
		{Path: f2, Card: card2},
	}
	if err := BatchRename(files, "[name].png"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "A.png")); err != nil {
		t.Errorf("A.png not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "B.png")); err != nil {
		t.Errorf("B.png not found: %v", err)
	}
}
