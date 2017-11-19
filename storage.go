package main

import (
  	"net/http"
    "encoding/json"
    "github.com/gorilla/mux"
    "github.com/syndtr/goleveldb/leveldb"
)

// StatusContinue           = 100 // RFC 7231, 6.2.1
// StatusSwitchingProtocols = 101 // RFC 7231, 6.2.2
// StatusProcessing         = 102 // RFC 2518, 10.1
//
// StatusOK                   = 200 // RFC 7231, 6.3.1
// StatusCreated              = 201 // RFC 7231, 6.3.2
// StatusAccepted             = 202 // RFC 7231, 6.3.3
// StatusNonAuthoritativeInfo = 203 // RFC 7231, 6.3.4
// StatusNoContent            = 204 // RFC 7231, 6.3.5
// StatusResetContent         = 205 // RFC 7231, 6.3.6
// StatusPartialContent       = 206 // RFC 7233, 4.1
// StatusMultiStatus          = 207 // RFC 4918, 11.1
// StatusAlreadyReported      = 208 // RFC 5842, 7.1
// StatusIMUsed               = 226 // RFC 3229, 10.4.1
//
// StatusMultipleChoices  = 300 // RFC 7231, 6.4.1
// StatusMovedPermanently = 301 // RFC 7231, 6.4.2
// StatusFound            = 302 // RFC 7231, 6.4.3
// StatusSeeOther         = 303 // RFC 7231, 6.4.4
// StatusNotModified      = 304 // RFC 7232, 4.1
// StatusUseProxy         = 305 // RFC 7231, 6.4.5
//
// StatusTemporaryRedirect = 307 // RFC 7231, 6.4.7
// StatusPermanentRedirect = 308 // RFC 7538, 3
//
// StatusBadRequest                   = 400 // RFC 7231, 6.5.1
// StatusUnauthorized                 = 401 // RFC 7235, 3.1
// StatusPaymentRequired              = 402 // RFC 7231, 6.5.2
// StatusForbidden                    = 403 // RFC 7231, 6.5.3
// StatusNotFound                     = 404 // RFC 7231, 6.5.4
// StatusMethodNotAllowed             = 405 // RFC 7231, 6.5.5
// StatusNotAcceptable                = 406 // RFC 7231, 6.5.6
// StatusProxyAuthRequired            = 407 // RFC 7235, 3.2
// StatusRequestTimeout               = 408 // RFC 7231, 6.5.7
// StatusConflict                     = 409 // RFC 7231, 6.5.8
// StatusGone                         = 410 // RFC 7231, 6.5.9
// StatusLengthRequired               = 411 // RFC 7231, 6.5.10
// StatusPreconditionFailed           = 412 // RFC 7232, 4.2
// StatusRequestEntityTooLarge        = 413 // RFC 7231, 6.5.11
// StatusRequestURITooLong            = 414 // RFC 7231, 6.5.12
// StatusUnsupportedMediaType         = 415 // RFC 7231, 6.5.13
// StatusRequestedRangeNotSatisfiable = 416 // RFC 7233, 4.4
// StatusExpectationFailed            = 417 // RFC 7231, 6.5.14
// StatusTeapot                       = 418 // RFC 7168, 2.3.3
// StatusUnprocessableEntity          = 422 // RFC 4918, 11.2
// StatusLocked                       = 423 // RFC 4918, 11.3
// StatusFailedDependency             = 424 // RFC 4918, 11.4
// StatusUpgradeRequired              = 426 // RFC 7231, 6.5.15
// StatusPreconditionRequired         = 428 // RFC 6585, 3
// StatusTooManyRequests              = 429 // RFC 6585, 4
// StatusRequestHeaderFieldsTooLarge  = 431 // RFC 6585, 5
// StatusUnavailableForLegalReasons   = 451 // RFC 7725, 3
//
// StatusInternalServerError           = 500 // RFC 7231, 6.6.1
// StatusNotImplemented                = 501 // RFC 7231, 6.6.2
// StatusBadGateway                    = 502 // RFC 7231, 6.6.3
// StatusServiceUnavailable            = 503 // RFC 7231, 6.6.4
// StatusGatewayTimeout                = 504 // RFC 7231, 6.6.5
// StatusHTTPVersionNotSupported       = 505 // RFC 7231, 6.6.6
// StatusVariantAlsoNegotiates         = 506 // RFC 2295, 8.1
// StatusInsufficientStorage           = 507 // RFC 4918, 11.5
// StatusLoopDetected                  = 508 // RFC 5842, 7.2
// StatusNotExtended                   = 510 // RFC 2774, 7
// StatusNetworkAuthenticationRequired = 511 // RFC 6585, 6

func (a *App) get(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    if vars["key"] == "" {
        respondWithError(w, http.StatusBadRequest, "Invalid key")
        return
    }

    db, err := leveldb.OpenFile(*dbpath, nil)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error reading the database")
        return
    }

    data, err := db.Get([]byte(vars["key"]), nil)
    if err != nil {
        respondWithError(w, http.StatusNotFound, "Item not found")
        return
    }

    i := item{Key: vars["key"], Value: string(data[:])}

    respondWithJSON(w, http.StatusOK, i)

    defer db.Close()
}

func (a *App) exists(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    if vars["key"] == "" {
        respondWithError(w, http.StatusBadRequest, "Invalid key")
        return
    }

    db, err := leveldb.OpenFile(*dbpath, nil)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error reading the database")
        return
    }

    _, err = db.Get([]byte(vars["key"]), nil)
    if err != nil {
        respondWithError(w, http.StatusNotFound, "Item not found")
        return
    }

    respondWithJSON(w, http.StatusOK, "Item exists")

    defer db.Close()
}

func (a *App) free(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    if vars["key"] == "" {
        respondWithError(w, http.StatusBadRequest, "Invalid key")
        return
    }

    db, err := leveldb.OpenFile(*dbpath, nil)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error reading the database")
        return
    }

    _, err = db.Get([]byte(vars["key"]), nil)
    if err != nil {
        respondWithError(w, http.StatusOK, "The space is available")
        return
    }

    respondWithJSON(w, http.StatusExpectationFailed, "Item exists")

    defer db.Close()
}

func (a *App) set(w http.ResponseWriter, r *http.Request) {
    var f item
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&f); err != nil {
        respondWithError(w, http.StatusBadRequest, err.Error())
        return
    }
    defer r.Body.Close()

    db, err := leveldb.OpenFile(*dbpath, nil)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error reading the database")
        return
    }

    err = db.Put([]byte(f.Key), []byte(f.Value), nil)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error writing the data")
        return
    }

    respondWithJSON(w, http.StatusCreated, f)

    defer db.Close()
}

func (a *App) del(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    if vars["key"] == "" {
        respondWithError(w, http.StatusBadRequest, "Invalid key")
        return
    }

    db, err := leveldb.OpenFile(*dbpath, nil)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error reading the database")
        return
    }

    err = db.Delete([]byte(vars["key"]), nil)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error deleting the data")
        return
    }

    respondWithJSON(w, http.StatusOK, "Item deleted")

    defer db.Close()
}
