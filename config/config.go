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
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/Sirupsen/logrus"

	"github.com/prometheus/common/log"

	"time"

	"github.com/google/gopacket/pcap"
)

var (
	device       = flag.String("i", "", "interface")
	snaplen      = flag.Int("s", 65535, "snaplen")
	help         = flag.Bool("h", false, "help")
	timeout      = flag.Int("t", int(pcap.BlockForever), "timeout")
	protocol     = flag.String("protocol", "http", "network protocol")
	expr         = flag.String("expr", "port 5000", "PCAP BPFFilter Rule")
	udpIP        = flag.String("server-host", "127.0.0.1", "udp server host ")
	udpPort      = flag.Int("server-port", 6666, "udp server port ")
	statsdServer = flag.String("statsd-server", "127.0.0.1:9125", "statsd server address")
)

//PCAPOption 抓包相关配置
type PCAPOption struct {
	Device  string
	Snaplen int
	HexDump bool
	Help    bool
	TimeOut time.Duration
}

//Option 主配置
type Option struct {
	PCAPOption
	StatsdServer   string
	UDPIP          string
	UDPPort        int
	SendCount      int
	Close          chan struct{}
	DiscoverConfig *DiscoverConfig
}

//Flagparse 解析参数
func Flagparse() *Option {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [ -i interface ] [ -s snaplen ] [ -h show usage] [ -t timeout] [ expression ] \n", os.Args[0])
		os.Exit(1)
	}
	flag.Parse()
	pcapOption := PCAPOption{
		Device:  *device,
		Snaplen: *snaplen,
		Help:    *help,
		TimeOut: time.Duration(*timeout),
	}
	option := &Option{
		PCAPOption:     pcapOption,
		UDPIP:          *udpIP,
		UDPPort:        *udpPort,
		StatsdServer:   *statsdServer,
		DiscoverConfig: GetDiscoverConfig(),
	}
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

//GetDiscoverConfig 获取配置信息
func GetDiscoverConfig() *DiscoverConfig {
	var disc DiscoverConfig
	discoverAddr := os.Getenv("DISCOVER_URL")
	if discoverAddr == "" {
		getBaseConfig(&disc)
		return &disc
	}
	res, err := http.Get(discoverAddr)
	if err != nil {
		logrus.Errorf("get discover configs error,%s", err.Error())
		getBaseConfig(&disc)
		return &disc
	}
	if res.Body != nil {
		defer res.Body.Close()
		if err := json.NewDecoder(res.Body).Decode(&disc); err != nil {
			logrus.Errorf("get discover configs error,%s", err.Error())
			getBaseConfig(&disc)
			return &disc
		}
	}
	return &disc
}

func getBaseConfig(disc *DiscoverConfig) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	protocol := os.Getenv("PROTOCOL")
	if port == 0 {
		port = 5000
	}
	if protocol == "" {
		protocol = "http"
	}
	disc.Ports = append(disc.Ports, Port{Port: port, Protocol: protocol})
}

//DiscoverConfig 配置发现
type DiscoverConfig struct {
	Ports []Port `json:"base_ports"`
}

//Port 端口信息
type Port struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}
