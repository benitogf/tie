package main

import (
	"log"
	"os"
)

var stdout = log.New(os.Stdout, "[pasticho] ", 0)
var stderr = log.New(os.Stderr, "[pasticho] ", 0)
