package opencodeadapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// JSONC is normalized for validation only; transaction writes preserve the
// caller's original bytes. No lossless config merger or evaluation is attempted.
func parseConfig(data []byte) (map[string]any, error) {
	if len(data) == 0 || len(data) > MaxInput {
		return nil, fmt.Errorf("config input exceeds limit or is empty")
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	clean := make([]byte, 0, len(data))
	quoted, escape := false, false
	for i := 0; i < len(data); i++ {
		b := data[i]
		if quoted {
			clean = append(clean, b)
			if escape {
				escape = false
			} else if b == '\\' {
				escape = true
			} else if b == '"' {
				quoted = false
			}
			continue
		}
		if b == '"' {
			quoted = true
			clean = append(clean, b)
			continue
		}
		if b == '/' && i+1 < len(data) && data[i+1] == '/' {
			for i < len(data) && data[i] != '\n' {
				i++
			}
			clean = append(clean, '\n')
			continue
		}
		if b == '/' && i+1 < len(data) && data[i+1] == '*' {
			i += 2
			for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
				i++
			}
			if i+1 >= len(data) {
				return nil, fmt.Errorf("unterminated config comment")
			}
			i++
			clean = append(clean, ' ')
			continue
		}
		clean = append(clean, b)
	}
	if quoted {
		return nil, fmt.Errorf("unterminated config string")
	}
	quoted, escape = false, false
	for i, b := range clean {
		if quoted {
			if escape {
				escape = false
			} else if b == '\\' {
				escape = true
			} else if b == '"' {
				quoted = false
			}
			continue
		}
		if b == '"' {
			quoted = true
			continue
		}
		if b == ',' {
			j := i + 1
			for j < len(clean) && strings.ContainsRune(" \t\r\n", rune(clean[j])) {
				j++
			}
			if j < len(clean) && (clean[j] == '}' || clean[j] == ']') {
				clean[i] = ' '
			}
		}
	}
	d := json.NewDecoder(bytes.NewReader(clean))
	d.UseNumber()
	value, err := readJSON(d, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid config JSON/duplicate field")
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("config must contain one object")
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config object required")
	}
	if sensitive(m) {
		return nil, fmt.Errorf("sensitive configuration input is not accepted")
	}
	return m, nil
}

func readJSON(d *json.Decoder, depth int) (any, error) {
	if depth > 32 {
		return nil, fmt.Errorf("JSON nesting limit")
	}
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			m := map[string]any{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, err
				}
				s, ok := key.(string)
				if !ok {
					return nil, fmt.Errorf("key required")
				}
				if _, exists := m[s]; exists {
					return nil, fmt.Errorf("duplicate key")
				}
				v, err := readJSON(d, depth+1)
				if err != nil {
					return nil, err
				}
				m[s] = v
			}
			_, err = d.Token()
			return m, err
		case '[':
			var values []any
			for d.More() {
				v, err := readJSON(d, depth+1)
				if err != nil {
					return nil, err
				}
				values = append(values, v)
			}
			_, err = d.Token()
			return values, err
		default:
			return nil, fmt.Errorf("unexpected delimiter")
		}
	}
	return t, nil
}

func sensitive(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for key, value := range x {
			key = strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key))
			switch key {
			case "apikey", "password", "token", "accesstoken", "refreshtoken", "secret", "secrets", "credential", "credentials", "authorization", "cookie", "headers":
				return true
			}
			if sensitive(value) {
				return true
			}
		}
	case []any:
		for _, value := range x {
			if sensitive(value) {
				return true
			}
		}
	case string:
		return strings.Contains(x, "Bearer ") || strings.Contains(x, "sk-or-") || strings.Contains(x, "sk-proj-")
	}
	return false
}
