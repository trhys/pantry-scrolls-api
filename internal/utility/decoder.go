package utility

import (
	"encoding/json"
	"net/http"
)

func DecodeRequest(w http.ResponseWriter, r *http.Request, maxBytes int64, T any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(T)
	if err != nil {
		return err
	}

	return nil
}
