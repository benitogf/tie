package main

import (
    "flag"
    "os"
    "log"
)

var address = flag.String("address", "localhost:9060", "http service address")
var ldbpath = flag.String("ldbpath", "storage/db", "leveldb storage path")
var authkey = flag.String("authkey", "my-secret-key", "secret key for auth key")
var authname = flag.String("authname", "Pasticho", "user name")
var authid = flag.String("authid", "000", "user id")
var authaccount = flag.String("authaccount", "pasticho", "user account name(login)")
var authpassword = flag.String("authpassword", "202cb962ac59075b964b07152d234b70", "user password")

func main() {
    flag.Parse()
    log.SetFlags(0)
    app := App{}
    app.Initialize(
            os.Getenv("PASTICHO_DB_USER"),
            os.Getenv("PASTICHO_DB_PASSWORD"),
            os.Getenv("PASTICHO_DB_NAME"))
    app.Run(address)
}
