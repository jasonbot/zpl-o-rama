package main

import (
	"flag"

	zplorama "github.com/jasonbot/zpl-o-rama/v1"
)

func main() {
	var listenport int
	var printerDialAddress string
	var printerServiceHost string

	flag.StringVar(&printerServiceHost, "servicehost", zplorama.Config.PrintserviceHost, "Address to bind to")
	flag.IntVar(&listenport, "listenport", 5491, "the port to listen on (bound to 127.0.0.1)")
	flag.StringVar(&printerDialAddress, "printeraddress", "192.168.1.1:9100", "Address of the Zebra printer on the network")
	flag.Parse()

	zplorama.RunPrintServer(printerServiceHost, listenport, printerDialAddress)
}
