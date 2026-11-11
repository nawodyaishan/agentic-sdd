package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

// ParseConfig reads at most 1 MiB and strictly decodes one schema-v1 document.
// It rejects oversize input, invalid UTF-8, duplicate object keys, unknown
// fields, trailing data and every schema/validation failure. Diagnostics name
// fields and instance IDs only; they never echo the document body.
func ParseConfig(r io.Reader) (*Config, error) {
	if r == nil {
		return nil, configError("input is nil")
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxConfigBytes+1))
	if err != nil {
		return nil, configError("cannot read config: %v", err)
	}
	if len(data) > MaxConfigBytes {
		return nil, configError("config exceeds %d bytes", MaxConfigBytes)
	}
	return parseConfigBytes(data)
}

func parseConfigBytes(data []byte) (*Config, error) {
	if !utf8.Valid(data) {
		return nil, configError("config is not valid UTF-8")
	}
	if err := detectDuplicateKeys(data); err != nil {
		return nil, configError("%v", err)
	}
	cfg, err := decodeStrict(data)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func decodeStrict(data []byte) (*Config, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, configError("%v", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, configError("trailing data after config document")
	}
	return &cfg, nil
}

// detectDuplicateKeys walks the JSON token stream and tracks keys separately
// for every object level. encoding/json keeps the last duplicate silently, so
// this pass must run before the strict struct decode.
func detectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	if err := walkJSONValue(dec, tok, 0); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected data after top-level value")
		}
		return fmt.Errorf("invalid JSON: %v", err)
	}
	return nil
}

func walkJSONValue(dec *json.Decoder, tok json.Token, depth int) error {
	if depth > 32 {
		return errors.New("JSON nesting limit exceeded")
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}
			key, ok := keyTok.(string)
			if !ok {
				return errors.New("JSON object key must be a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate key %q", key)
			}
			seen[key] = struct{}{}
			valueTok, err := dec.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}
			if err := walkJSONValue(dec, valueTok, depth+1); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("invalid JSON: %v", err)
		}
		return nil
	case '[':
		for dec.More() {
			valueTok, err := dec.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}
			if err := walkJSONValue(dec, valueTok, depth+1); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("invalid JSON: %v", err)
		}
		return nil
	default:
		return fmt.Errorf("invalid JSON delimiter %q", delim)
	}
}
