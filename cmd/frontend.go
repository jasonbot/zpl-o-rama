package main

import (
	"flag"
	"fmt"

	zplorama "github.com/jasonbot/zpl-o-rama/v1"
)

func main() {
	var listenport int
	var printerurl string
	var configFile string

	flag.IntVar(&listenport, "listenport", zplorama.Config.FrontendPort, "the port to listen on (bound to 0.0.0.0)")
	flag.StringVar(&printerurl, "printerurl", fmt.Sprintf("http://%s:%v", zplorama.Config.PrintserviceHost, zplorama.Config.PrintservicePort), "the url to the printing service")
	flag.StringVar(&configFile, "configfile", "", "Path to config.json")
	flag.Parse()

	if configFile != "" {
		zplorama.LoadConfig(configFile)
	}

	zplorama.RunFrontendServer(listenport, printerurl)
}
