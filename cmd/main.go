// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"tcm/config"
	"tcm/net"
	"time"

	_ "net/http/pprof"

	"github.com/prometheus/common/log"
)

func main() {
	option := config.Flagparse()
	if option.Help {
		flag.Usage()
	}
	//多端口支持
	for _, port := range option.DiscoverConfig.Ports {
		go func(port config.Port) {
			decode := net.FindDecode(option, port)
			if decode == nil {
				log.Errorf("protocol %s can not support or decode manange create error", port.Protocol)
				os.Exit(1)
			}
			netUtil := net.CreateUtil(port, decode, option)
			if r := netUtil.Pcap(); r != 0 {
				os.Exit(r)
			}
		}(port)
	}
	go httpListener()
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case <-term:
		log.Warn("Received SIGTERM, exiting gracefully...")
		// case err := <-errch:
		// 	log.Error(err.Error())
	}
	close(option.Close)
	time.Sleep(time.Second * 4)
	log.Info("See you next time!")
}

func httpListener() {
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello word"))
	})
	http.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Errorf("listen http port 5000 error %s", err.Error())
	}
}
