package main

import (
	"flag"
)

func main() {
	var listenport int
	var printerurl string

	flag.IntVar(&listenport, "listenport", 5489, "the port to listen on (bound to 0.0.0.0)")
	flag.StringVar(&printerurl, "printerurl", "http://127.0.0.1:5491", "the url to the printing service")
	flag.Parse()
}
