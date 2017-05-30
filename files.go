package main

import (
    "strconv"
  	"net/http"
    "database/sql"
  	"encoding/json"
    "github.com/gorilla/mux"
    _ "github.com/lib/pq"
)

func (app *App) createFile(w http.ResponseWriter, r *http.Request) {
    var f file
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&f); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }
    defer r.Body.Close()

    if err := f.createFile(app.DB); err != nil {
        respondWithError(w, http.StatusInternalServerError, err.Error())
        return
    }

    respondWithJSON(w, http.StatusCreated, f)
}

func (a *App) getFile(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"],10,64)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid file ID")
        return
    }

    f := file{ID: id}
    if err := f.getFile(a.DB); err != nil {
        switch err {
        case sql.ErrNoRows:
            respondWithError(w, http.StatusNotFound, "File not found")
        default:
            respondWithError(w, http.StatusInternalServerError, err.Error())
        }
        return
    }

    respondWithJSON(w, http.StatusOK, f)
}

func (a *App) deleteFile(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"],10,64)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid file ID")
        return
    }

    f := file{ID: id}
    if err := f.deleteFile(a.DB); err != nil {
        respondWithError(w, http.StatusInternalServerError, err.Error())
        return
    }

    respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}
