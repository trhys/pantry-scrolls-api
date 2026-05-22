package main

import (
	"encoding/json"
	"log"
	"net/http"
)


func respondFail(w http.ResponseWriter, code int, msg string, err error) {
	log.Print(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil  {
		log.Printf("Failed to marshal json in response: %v", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
