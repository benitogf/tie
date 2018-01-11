package main

import (
    "log"
    "os"
)

var stdout = log.New(os.Stdout, "[tie] ", 0)
var stderr = log.New(os.Stderr, "[tie] ", 0)
