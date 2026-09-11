package service

import "encoding/json"

// requestWithExtra adds future Core request fields without replacing fields
// already emitted by the typed request. Omitted optional fields can be supplied
// through extra. Query parameters remain separate from the JSON body.
func requestWithExtra(data any, extra map[string]any) (any, error) {
	if len(extra) == 0 {
		return data, nil
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	for key, value := range extra {
		if _, exists := fields[key]; exists {
			continue
		}
		encoded, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
		fields[key] = encoded
	}
	return fields, nil
}
