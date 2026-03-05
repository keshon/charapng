package charapng

import "encoding/json"

// Card holds the decoded character card data extracted from a PNG file.
// Keyword is the tEXt chunk key under which the data was found (e.g. "chara", "ccv3").
// JSON contains the raw decoded payload and can be unmarshaled into any structure.
type Card struct {
	Keyword string
	JSON    json.RawMessage
}

// Map unmarshals the card's JSON payload into a generic map.
// Useful for quick inspection without a typed struct.
func (c *Card) Map() (map[string]any, error) {
	var m map[string]any

	if err := json.Unmarshal(c.JSON, &m); err != nil {
		return nil, err
	}

	return m, nil
}
