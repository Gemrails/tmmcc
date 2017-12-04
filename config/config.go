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

package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/prometheus/common/log"

	"time"

	"github.com/google/gopacket/pcap"
)

var (
	device   = flag.String("i", "", "interface")
	snaplen  = flag.Int("s", 65535, "snaplen")
	help     = flag.Bool("h", false, "help")
	timeout  = flag.Int("t", int(pcap.BlockForever), "timeout")
	protocol = flag.String("p", "http", "network protocol")
	udpIP    = flag.String("server-host", "172.30.42.1", "udp server host ")
	udpPort  = flag.Int("server-port", 6666, "udp server port ")
)

//PCAPOption 抓包相关配置
type PCAPOption struct {
	Device   string
	Snaplen  int
	HexDump  bool
	Help     bool
	TimeOut  time.Duration
	Protocol string
	Expr     string
}

//Option 主配置
type Option struct {
	PCAPOption
	UDPIP     string
	UDPPort   int
	SendCount int
	Close     chan struct{}
}

//Flagparse 解析参数
func Flagparse() *Option {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [ -i interface ] [ -s snaplen ] [ -h show usage] [ -t timeout] [ -c zmq client number ] [ -zmq zmq server url ] [ -p network protocol] [ expression ] \n", os.Args[0])
		os.Exit(1)
	}
	flag.Parse()
	pcapOption := PCAPOption{
		Device:   *device,
		Snaplen:  *snaplen,
		Help:     *help,
		TimeOut:  time.Duration(*timeout),
		Protocol: *protocol,
	}
	option := &Option{
		PCAPOption: pcapOption,
		UDPIP:      *udpIP,
		UDPPort:    *udpPort,
	}
	//过滤语法，同tcpdump
	expr := ""
	if len(flag.Args()) > 0 {
		expr = flag.Arg(0)
	}
	if expr == "" {
		Port := os.Getenv("PORT")
		if Port == "" {
			Port = "5000"
		}
		expr = "port " + Port
	}
	option.Expr = expr

	if option.Device == "" {
		devs, err := pcap.FindAllDevs()
		if err != nil {
			log.Errorln(os.Stderr, "tcpdump: couldn't find any devices:", err)
		}
		if 0 == len(devs) {
			flag.Usage()
		}
		option.Device = devs[0].Name
	}
	option.Close = make(chan struct{})
	return option
}
