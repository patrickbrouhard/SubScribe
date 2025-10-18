package subtitles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ParseJSON3Bytes parse un blob JSON ([]byte) et retourne la structure rawJSON3.
//
// Utilise json.Decoder en lecture depuis un bytes.Reader quand les données sont
// déjà en présentes 100% en mémoire : adapté aux fichiers pas trop volumineux
func ParseJSON3Bytes(b []byte) (rawJSON3, error) {
	var raw rawJSON3
	if len(b) == 0 {
		return raw, fmt.Errorf("ParseJSON3Bytes: empty input")
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	// Ne pas appeler DisallowUnknownFields() car le JSON contient souvent des champs
	// inutiles/non mappés — on veut ignorer proprement ces champs.
	if err := dec.Decode(&raw); err != nil {
		return raw, fmt.Errorf("ParseJSON3Bytes: decode error: %w", err)
	}
	return raw, nil
}

// ParseJSON3Reader parse depuis un io.Reader (utile si on veut décoder depuis un flux)
func ParseJSON3Reader(r io.Reader) (rawJSON3, error) {
	var raw rawJSON3
	dec := json.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return raw, fmt.Errorf("ParseJSON3Reader: decode error: %w", err)
	}
	return raw, nil
}
