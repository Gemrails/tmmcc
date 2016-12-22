package main

import (
	"flag"
	"os"
	"tcm/config"
	"tcm/net"
)

func main() {
	option := config.Flagparse()
	if option.Help {
		flag.Usage()
	}
	if option.Protocol == "http" {
		stop := make(chan struct{})
		defer close(stop)
		net.CreateHTTPManager(stop)
	}
	netUtil := net.Util{
		Option: option,
	}
	if r := netUtil.Pcap(); r != 0 {
		os.Exit(r)
	}
}
