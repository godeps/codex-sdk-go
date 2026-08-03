package protocol

import "encoding/json"

type UnknownUnionValue struct {
	Raw json.RawMessage
}

func (v *UnknownUnionValue) UnmarshalJSON(data []byte) error {
	if v == nil {
		return nil
	}
	v.Raw = append(json.RawMessage(nil), data...)
	return nil
}

func (v UnknownUnionValue) MarshalJSON() ([]byte, error) {
	if len(v.Raw) == 0 {
		return []byte("null"), nil
	}
	return append(json.RawMessage(nil), v.Raw...), nil
}

func unionDiscriminator(data []byte, field string) (string, bool) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false
	}
	raw, ok := payload[field]
	if !ok {
		return "", false
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false
	}
	return out, true
}

func jsonLooksObject(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonLooksString(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '"':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonStringIn(data []byte, values ...string) bool {
	var out string
	if err := json.Unmarshal(data, &out); err != nil {
		return false
	}
	for _, value := range values {
		if out == value {
			return true
		}
	}
	return false
}

func jsonLooksArray(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '[':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonLooksBool(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case 't', 'f':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonLooksNumber(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonHasKeys(data []byte, keys ...string) bool {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	for _, key := range keys {
		if _, ok := payload[key]; !ok {
			return false
		}
	}
	return true
}

func jsonHasField(data []byte, field string) bool {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	_, ok := payload[field]
	return ok
}
