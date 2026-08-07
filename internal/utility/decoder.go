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

// This helper is to check slices in our request bodies that should not be empty
// such as ingredients
func CheckNilSlice(s [][]any) error {
	for _, i := range s {
		if len(i) == 0 {
			return fmt.Errorf("Empty slice in request body")
		}
	}

	return nil
}
