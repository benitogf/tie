package main

import (
    "flag"
    "os"
    "log"
)

var address = flag.String("address", "localhost:9060", "http service address")
var ldbpath = flag.String("ldbpath", "storage/db", "leveldb storage path")
var authkey = flag.String("authkey", "my-secret-key", "secret key for auth key")
// fixed admin
var adminname = flag.String("adminname", "Admin", "admin name")
var adminid = flag.String("adminid", "000", "admin id")
var adminaccount = flag.String("adminaccount", "admin", "admin account name(login)")
var adminpassword = flag.String("adminpassword", "202cb962ac59075b964b07152d234b70", "admin password")
// fixed user
var username = flag.String("username", "User", "user name")
var userid = flag.String("userid", "001", "user id")
var useraccount = flag.String("useraccount", "user", "user account name(login)")
var userpassword = flag.String("userpassword", "202cb962ac59075b964b07152d234b70", "user password")

func main() {
    flag.Parse()
    log.SetFlags(0)
    app := App{}
    app.Initialize(
            os.Getenv("TIE_DB_USER"),
            os.Getenv("TIE_DB_PASSWORD"),
            os.Getenv("TIE_DB_NAME"))
    app.Run(address)
}
