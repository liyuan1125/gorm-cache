package cache

import "encoding/json"

type DefaultJSONSerializer struct{}

func (d *DefaultJSONSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (d *DefaultJSONSerializer) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
