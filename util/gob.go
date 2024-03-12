package util

import (
	"encoding/gob"
)

func GobMarshal(v any) ([]byte, error) {
	var buffer ByteBuffer
	buffer.InitCap(128)
	err := gob.NewEncoder(&buffer).Encode(v)
	if err != nil {
		return nil, err
	}
	return buffer.All(), nil
}

func GobUnmarshal(data []byte, v any) error {
	var buffer ByteBuffer
	buffer.InitBytes(data)
	return gob.NewDecoder(&buffer).Decode(v)
}
