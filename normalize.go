package charapng

import (
	"encoding/json"
	"fmt"
)

// ExtractCardMetadata opens the PNG at path, decodes the embedded card payload,
// detects spec version (v1/v2/v3), and returns a normalized Card.
// Returns an error if the file cannot be read or the payload is not valid card JSON.
func ExtractCardMetadata(path string) (*Card, error) {
	raw, err := DecodeFile(path)
	if err != nil {
		return nil, err
	}
	return normalizeFromRaw(raw)
}

// normalizeFromRaw converts a RawCard into a normalized Card by detecting spec and parsing.
func normalizeFromRaw(raw *RawCard) (*Card, error) {
	if raw == nil {
		return nil, fmt.Errorf("charapng: nil raw card")
	}
	var m map[string]any
	if err := json.Unmarshal(raw.JSON, &m); err != nil {
		return nil, fmt.Errorf("charapng: invalid card JSON: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("charapng: empty card JSON")
	}

	card := &Card{Raw: m}

	// Detect spec: v2 has "spec": "chara_card_v2" and "data"
	if spec, _ := m["spec"].(string); spec == "chara_card_v2" {
		if data, ok := m["data"].(map[string]any); ok {
			card.SpecVersion = "v2"
			card.Name = str(data, "name")
			card.Description = str(data, "description")
			card.Personality = str(data, "personality")
			card.Scenario = str(data, "scenario")
			card.FirstMessage = str(data, "first_mes")
			card.ExampleDialogue = str(data, "mes_example")
			card.Notes = str(data, "creator_notes")
			card.Tags = strSlice(data, "tags")
			card.Creator = str(data, "creator") // if present in v2
			_ = str(data, "system_prompt")      // v2 field, not in normalized model
			return card, nil
		}
	}

	// v3: often has "spec" or was found under key "ccv3"; structure can mirror v2 or have extensions
	if spec, _ := m["spec"].(string); spec != "" && spec != "chara_card_v2" {
		card.SpecVersion = "v3"
		// v3 may have top-level or nested data
		if data, ok := m["data"].(map[string]any); ok {
			card.Name = str(data, "name")
			card.Description = str(data, "description")
			card.Personality = str(data, "personality")
			card.Scenario = str(data, "scenario")
			card.FirstMessage = str(data, "first_mes")
			card.ExampleDialogue = str(data, "mes_example")
			card.Notes = str(data, "creator_notes")
			card.Tags = strSlice(data, "tags")
			card.Creator = str(data, "creator")
		} else {
			fillFromMap(card, m)
		}
		return card, nil
	}

	// v1: flat object with name, description, first_mes, etc.
	if _, hasName := m["name"]; hasName || m["description"] != nil || m["first_mes"] != nil {
		card.SpecVersion = "v1"
		fillFromMap(card, m)
		return card, nil
	}

	// Fallback: treat as v1 and fill what we can
	card.SpecVersion = "v1"
	fillFromMap(card, m)
	return card, nil
}

func fillFromMap(card *Card, m map[string]any) {
	card.Name = str(m, "name")
	card.Description = str(m, "description")
	card.Personality = str(m, "personality")
	card.Scenario = str(m, "scenario")
	card.FirstMessage = str(m, "first_mes")
	card.ExampleDialogue = str(m, "mes_example")
	card.Notes = str(m, "creator_notes")
	card.Tags = strSlice(m, "tags")
	card.Creator = str(m, "creator")
}

func str(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func strSlice(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, x := range s {
			if t, ok := x.(string); ok {
				out = append(out, t)
			}
		}
		return out
	}
	return nil
}
