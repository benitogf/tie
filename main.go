package main

import (
    "flag"
	"os"
    "log"
)

var addr = flag.String("addr", "localhost:9060", "http service address")
var dbpath = flag.String("dbpath", "storage/db", "http service address")

func main() {

	flag.Parse()
	log.SetFlags(0)
	app := App{}
	app.Initialize(
			os.Getenv("PASTICHO_DB_USER"),
			os.Getenv("PASTICHO_DB_PASSWORD"),
			os.Getenv("PASTICHO_DB_NAME"))
	app.Run(addr)
}
