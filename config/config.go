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
	//同时发送消息的协程数量
	sendcount = flag.Int("c", 1, "the number of send zmq")
	zmqurl    = flag.String("zmq", "127.0.0.1", "zmq server url ")
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
	ZMQURL    string
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
		ZMQURL:     *zmqurl,
		SendCount:  *sendcount,
	}
	//过滤语法，同tcpdump
	expr := ""
	if len(flag.Args()) > 0 {
		expr = flag.Arg(0)
	}
	if expr == "" {
		Port := os.Getenv("PORT")
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
