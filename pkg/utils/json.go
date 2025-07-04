package utils

import (
	"encoding/json"
)

// MustMarshal marshals the given value into JSON and panics if an error occurs
func MustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
