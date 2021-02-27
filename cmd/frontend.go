package main

import (
	"flag"
	"fmt"

	zplorama "github.com/jasonbot/zpl-o-rama/v1"
)

func main() {
	var listenport int
	var printerurl string

	flag.IntVar(&listenport, "listenport", zplorama.Config.FrontendPort, "the port to listen on (bound to 0.0.0.0)")
	flag.StringVar(&printerurl, "printerurl", fmt.Sprintf("http://%s:%v", zplorama.Config.PrintserviceHost, zplorama.Config.PrintservicePort), "the url to the printing service")
	flag.Parse()

	zplorama.RunFrontendServer(listenport, printerurl)
}
