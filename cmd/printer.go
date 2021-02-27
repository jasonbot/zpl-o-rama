package main

import (
	"flag"

	zplorama "github.com/jasonbot/zpl-o-rama/v1"
)

func main() {
	var listenport int
	var printerDialAddress string

	flag.IntVar(&listenport, "listenport", 5491, "the port to listen on (bound to 127.0.0.1)")
	flag.StringVar(&printerDialAddress, "printeraddress", "192.168.1.1:90", "Address of the printer on the network")

	zplorama.RunPrintServer(listenport, printerDialAddress)
}
