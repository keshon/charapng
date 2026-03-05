# charapng

`charapng` is a small Go library for extracting character card data embedded inside PNG images.

Many roleplay/AI character cards distribute metadata by embedding a Base64-encoded JSON document inside PNG `tEXt` chunks (commonly under the `chara` key). This package provides a simple and idiomatic way to locate, decode, and access that data.

The package focuses on:

* reading PNG chunks directly
* locating character metadata fields
* decoding Base64 payloads
* returning the raw JSON for further processing

It is designed as a reusable library and can easily be embedded into CLI tools, bots, or web services that need to process character cards.

## Features

* Reads PNG files without external dependencies
* Extracts `tEXt` metadata chunks
* Detects character card payloads
* Decodes Base64 metadata automatically
* Returns raw JSON or parsed structures

## Installation

```
go get github.com/keshon/charapng
```

## Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/keshon/charapng"
)

func main() {
	card, err := charapng.DecodeFile("card.png")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Keyword:", card.Keyword)
	fmt.Println(string(card.JSON))
}
```

## Typical Use Case

Character cards used in AI roleplay platforms often bundle:

* avatar image
* metadata
* scenario
* prompts
* character description

All inside a single PNG file.

`charapng` extracts that hidden metadata so it can be processed programmatically.

## Returned Structure

```go
type Card struct {
	Keyword string
	JSON    []byte
}
```

`JSON` contains the decoded character card data which can be unmarshaled into your own structures.

## CLI Example

A simple CLI tool can iterate through a folder of cards and print the decoded metadata.

```
cards/
  card1.png
  card2.png
```

```
go run .
```

## License

MIT
