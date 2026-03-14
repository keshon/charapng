package charapng

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

// writeMinimalPNG writes a minimal PNG to path with one tEXt chunk (keyword\0value).
func writeMinimalPNG(path, keyword, value string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sig := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	if _, err := f.Write(sig); err != nil {
		return err
	}
	ihdrData := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdrData[0:4], 1)
	binary.BigEndian.PutUint32(ihdrData[4:8], 1)
	ihdrData[8] = 8
	ihdrData[9] = 2
	if err := writeChunk(f, "IHDR", ihdrData); err != nil {
		return err
	}
	if err := writeChunk(f, "tEXt", []byte(keyword+"\x00"+value)); err != nil {
		return err
	}
	if err := writeChunk(f, "IEND", nil); err != nil {
		return err
	}
	return nil
}

func writeChunk(f *os.File, chunkType string, data []byte) error {
	payload := append([]byte(chunkType), data...)
	crc := crc32.ChecksumIEEE(payload)
	if err := binary.Write(f, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	if _, err := f.Write([]byte(chunkType)); err != nil {
		return err
	}
	if len(data) > 0 {
		if _, err := f.Write(data); err != nil {
			return err
		}
	}
	return binary.Write(f, binary.BigEndian, crc)
}

func TestExtractV1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v1.png")
	v1JSON := `{"name":"TestChar","description":"A test","personality":"kind","scenario":"test scenario","first_mes":"Hello","mes_example":"User: hi\nChar: hello"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(v1JSON))
	if err := writeMinimalPNG(path, "chara", encoded); err != nil {
		t.Fatal(err)
	}
	card, err := ExtractCardMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if card.SpecVersion != "v1" {
		t.Errorf("SpecVersion = %q, want v1", card.SpecVersion)
	}
	if card.Name != "TestChar" {
		t.Errorf("Name = %q, want TestChar", card.Name)
	}
	if card.Description != "A test" {
		t.Errorf("Description = %q", card.Description)
	}
	if card.FirstMessage != "Hello" {
		t.Errorf("FirstMessage = %q", card.FirstMessage)
	}
}

func TestExtractV2(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v2.png")
	v2JSON := `{"spec":"chara_card_v2","data":{"name":"V2Char","description":"Desc","personality":"p","scenario":"s","first_mes":"Hi","mes_example":"","creator_notes":"notes","tags":["tag1","tag2"]}}`
	encoded := base64.StdEncoding.EncodeToString([]byte(v2JSON))
	if err := writeMinimalPNG(path, "chara", encoded); err != nil {
		t.Fatal(err)
	}
	card, err := ExtractCardMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if card.SpecVersion != "v2" {
		t.Errorf("SpecVersion = %q, want v2", card.SpecVersion)
	}
	if card.Name != "V2Char" {
		t.Errorf("Name = %q, want V2Char", card.Name)
	}
	if card.Notes != "notes" {
		t.Errorf("Notes = %q", card.Notes)
	}
	if len(card.Tags) != 2 || card.Tags[0] != "tag1" || card.Tags[1] != "tag2" {
		t.Errorf("Tags = %v", card.Tags)
	}
}

func TestExtractV3(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v3.png")
	// v3: spec present but not chara_card_v2; can have data or flat structure
	v3JSON := `{"spec":"chara_card_v3","data":{"name":"V3Char","description":"D","personality":"P","scenario":"S","first_mes":"F","mes_example":"E","tags":["a","b"]}}`
	encoded := base64.StdEncoding.EncodeToString([]byte(v3JSON))
	if err := writeMinimalPNG(path, "ccv3", encoded); err != nil {
		t.Fatal(err)
	}
	card, err := ExtractCardMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if card.SpecVersion != "v3" {
		t.Errorf("SpecVersion = %q, want v3", card.SpecVersion)
	}
	if card.Name != "V3Char" {
		t.Errorf("Name = %q, want V3Char", card.Name)
	}
	if len(card.Tags) != 2 || card.Tags[0] != "a" {
		t.Errorf("Tags = %v", card.Tags)
	}
}

// TestExtractFromTestdata runs ExtractCardMetadata on testdata/v1.png, v2.png, v3.png if present.
// Create them by running: GEN_TESTDATA=1 go test -run TestGenerateTestdata
func TestExtractFromTestdata(t *testing.T) {
	for name, wantName := range map[string]string{
		"testdata/v1.png": "TestChar",
		"testdata/v2.png": "V2Char",
		"testdata/v3.png": "V3Char",
	} {
		if _, err := os.Stat(name); err != nil {
			t.Skipf("testdata not found (run GEN_TESTDATA=1 go test -run TestGenerateTestdata to create)")
		}
		card, err := ExtractCardMetadata(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if card.Name != wantName {
			t.Errorf("%s: Name = %q, want %q", name, card.Name, wantName)
		}
	}
}

// TestGenerateTestdata creates testdata/v1.png, v2.png, v3.png. Run with GEN_TESTDATA=1.
func TestGenerateTestdata(t *testing.T) {
	if os.Getenv("GEN_TESTDATA") != "1" {
		t.Skip("set GEN_TESTDATA=1 to generate testdata files")
	}
	if err := os.MkdirAll("testdata", 0755); err != nil {
		t.Fatal(err)
	}
	v1JSON := `{"name":"TestChar","description":"A test","personality":"kind","scenario":"test scenario","first_mes":"Hello","mes_example":"User: hi\nChar: hello"}`
	if err := writeMinimalPNG("testdata/v1.png", "chara", base64.StdEncoding.EncodeToString([]byte(v1JSON))); err != nil {
		t.Fatal(err)
	}
	v2JSON := `{"spec":"chara_card_v2","data":{"name":"V2Char","description":"Desc","personality":"p","scenario":"s","first_mes":"Hi","mes_example":"","creator_notes":"notes","tags":["tag1","tag2"]}}`
	if err := writeMinimalPNG("testdata/v2.png", "chara", base64.StdEncoding.EncodeToString([]byte(v2JSON))); err != nil {
		t.Fatal(err)
	}
	v3JSON := `{"spec":"chara_card_v3","data":{"name":"V3Char","description":"D","personality":"P","scenario":"S","first_mes":"F","mes_example":"E","tags":["a","b"]}}`
	if err := writeMinimalPNG("testdata/v3.png", "ccv3", base64.StdEncoding.EncodeToString([]byte(v3JSON))); err != nil {
		t.Fatal(err)
	}
	t.Log("created testdata/v1.png, v2.png, v3.png")
}
