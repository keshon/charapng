package charapng

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type chunk struct {
	Type string
	Data []byte
}

var pngSignature = []byte{137, 80, 78, 71, 13, 10, 26, 10}

// maxChunkSize guards against malformed or malicious files that claim enormous chunk lengths.
const maxChunkSize = 256 * 1024 * 1024 // 256 MiB

func readPNGChunks(r io.Reader) ([]chunk, error) {
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, fmt.Errorf("read signature: %w", err)
	}
	if !bytes.Equal(sig, pngSignature) {
		return nil, errors.New("not a valid PNG file")
	}

	var out []chunk

	for {
		var length uint32
		if err := binary.Read(r, binary.BigEndian, &length); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errors.New("unexpected end of file: missing IEND chunk")
			}
			return nil, fmt.Errorf("read chunk length: %w", err)
		}

		if length > maxChunkSize {
			return nil, fmt.Errorf("chunk length %d exceeds maximum allowed size", length)
		}

		typ := make([]byte, 4)
		if _, err := io.ReadFull(r, typ); err != nil {
			return nil, fmt.Errorf("read chunk type: %w", err)
		}

		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("read chunk data: %w", err)
		}

		// Discard the 4-byte CRC.
		if _, err := io.CopyN(io.Discard, r, 4); err != nil {
			return nil, fmt.Errorf("read chunk crc: %w", err)
		}

		t := string(typ)
		out = append(out, chunk{Type: t, Data: data})

		if t == "IEND" {
			break
		}
	}

	return out, nil
}
