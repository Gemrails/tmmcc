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

package net

import (
	"fmt"
	"tcm/config"

	"github.com/prometheus/common/log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

//Util 网络抓包工具
type Util struct {
	Option *config.Option
	Port   config.Port
	Decode Decode
}

//CreateUtil 创建抓包器
func CreateUtil(Port config.Port, Decode Decode, Option *config.Option) *Util {
	return &Util{
		Option: Option,
		Port:   Port,
		Decode: Decode,
	}
}

//Pcap 抓包
func (n *Util) Pcap() int {
	devs, err := pcap.FindAllDevs()
	if err != nil {
		log.Errorln("tcpdump: couldn't find any devices:", err)
		return 1
	}
	for _, device := range devs {
		if handle, err := pcap.OpenLive(device.Name, int32(n.Option.Snaplen), true, n.Option.TimeOut); err != nil {
			log.With("error", err.Error()).Errorln("PCAP OpenLive Error.")
			return 1
		} else if err := handle.SetBPFFilter(fmt.Sprintf("port %d", n.Port.Port)); err != nil { // optional
			log.With("error", err.Error()).Errorln("PCAP SetBPFFilter Error.", fmt.Sprintf("port %d", n.Port.Port))
			return 1
		} else {
			log.Infof("Start listen the device %s %s ", device.Name, fmt.Sprintf("port %d", n.Port.Port))
			packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
			go func(close chan struct{}, h *pcap.Handle, deviceName string) {
				for {
					select {
					case packet := <-packetSource.Packets():
						n.handlePacket(packet) // Do something with a packet here.
					case <-close:
						log.Infof("stop listen the device %s.", deviceName)
						h.Close()
						return
					}
				}
			}(n.Option.Close, handle, device.Name)
		}
	}
	return 0
}

func (n *Util) handlePacket(packet gopacket.Packet) {
	app := packet.ApplicationLayer()
	if app != nil {
		//log.With("type", app.LayerType().String()).Infoln("Receive a application layer packet")
		//log.Infoln(packet.String())
		//fmt.Println(app.Payload())
		sd := &SourceData{
			Source:      app.Payload(),
			ReceiveDate: packet.Metadata().Timestamp,
		}
		tran := packet.TransportLayer()
		if tran != nil {
			src, dst := tran.TransportFlow().Endpoints()
			sd.SourcePoint = &src
			sd.TargetPoint = &dst
			if tran.LayerType().Contains(layers.LayerTypeTCP) {
				tcp := &layers.TCP{}
				err := tcp.DecodeFromBytes(tran.LayerContents(), gopacket.NilDecodeFeedback)
				if err != nil {
					log.With("error", err.Error()).Errorln("Decode bytes to TCP error")
				} else {
					sd.TCP = tcp
				}
			}
		}
		netL := packet.NetworkLayer()
		if netL != nil {
			src, dst := packet.NetworkLayer().NetworkFlow().Endpoints()
			sd.SourceHost = &src
			sd.TargetHost = &dst
		}
		n.Decode.Decode(sd)
	}
}
