package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"tcm/config"
	"tcm/net"
	"time"

	"github.com/prometheus/common/log"
)

func main() {
	option := config.Flagparse()
	if option.Help {
		flag.Usage()
	}
	if option.Protocol == "http" {
		err := net.CreateHTTPManager(option)
		if err != nil {
			log.With("error", err.Error()).Errorln("create http manager error.")
			os.Exit(0)
		}
	}
	netUtil := net.Util{
		Option: option,
	}
	if r := netUtil.Pcap(); r != 0 {
		os.Exit(r)
	}
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case <-term:
		log.Warn("Received SIGTERM, exiting gracefully...")
	}
	close(option.Close)
	time.Sleep(time.Second * 4)
	log.Info("See you next time!")
}
