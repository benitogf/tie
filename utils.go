package main

import (
    "net/http"
      "encoding/json"
    "strings"
    "strconv"
)

func respondWithError(w http.ResponseWriter, code int, message string) {
    respondWithJSON(w, code, map[string]string{"message": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    response, _ := json.Marshal(payload)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
}

func ToStrf(s interface{}) string {
    result := []string{}
    aux := s.([]int64)
    for i := 0; i < len(aux); i++ {
        result = append(result, strconv.FormatInt(aux[i], 10))
    }
    return "{"+strings.Join(result, ",")+"}"
}
