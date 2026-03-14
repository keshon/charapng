package charapng

import "encoding/json"

// RawCard holds the decoded character card data extracted from a PNG file.
// Keyword is the tEXt chunk key under which the data was found (e.g. "chara", "ccv3").
// JSON contains the raw decoded payload. Use Decode/DecodeFile to obtain RawCard;
// use ExtractCardMetadata to get a normalized Card.
type RawCard struct {
	Keyword string
	JSON    json.RawMessage
}

// Map unmarshals the raw card's JSON payload into a generic map.
func (c *RawCard) Map() (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(c.JSON, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Card is the normalized character card model (spec v1/v2/v3).
// All extractable text fields are unified here for display and rename templates.
type Card struct {
	SpecVersion string

	Name        string
	Description string
	Personality string
	Scenario    string

	FirstMessage    string
	ExampleDialogue string

	Tags    []string
	Creator string
	Notes   string

	Raw map[string]any
}
