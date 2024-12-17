package helper

import (
	"encoding/json"
	"net/http"
)

func ReadRequest(r *http.Request, result interface{}) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(result)
	PanicOnError(err)
}

func WriteResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Add("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	PanicOnError(err)
}
