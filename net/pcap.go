package net

import (
	"tcm/config"
	"time"

	"github.com/prometheus/common/log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

//Util 网络抓包工具
type Util struct {
	Option *config.PCAPOption
}

//Pcap 抓包
func (n *Util) Pcap() int {

	if handle, err := pcap.OpenLive(n.Option.Device, int32(n.Option.Snaplen), true, n.Option.TimeOut); err != nil {
		log.With("error", err.Error()).Errorln("PCAP OpenLive Error.")
		return 1
	} else if err := handle.SetBPFFilter(n.Option.Expr); err != nil { // optional
		log.With("error", err.Error()).Errorln("PCAP SetBPFFilter Error.", n.Option.Expr)
		return 1
	} else {
		log.Infoln("Start listen the device ", n.Option.Device)
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		for packet := range packetSource.Packets() {
			n.handlePacket(packet) // Do something with a packet here.
		}
	}
	return 0
}

func (n *Util) handlePacket(packet gopacket.Packet) {
	app := packet.ApplicationLayer()
	if app != nil {
		log.With("type", app.LayerType().String()).Infoln("Receive a application layer packet")
		go func() {
			sd := &SourceData{
				Source:      app.Payload(),
				ReceiveDate: time.Now(),
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
			decode := FindDecode(n.Option.Protocol)
			if decode != nil {
				decode.Decode(sd)
			} else {
				log.Debugf("%s protol can not be supported \n", n.Option.Protocol)
			}
		}()
	}

}
