package net

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

//SourceData 解码前数据
type SourceData struct {
	Source      []byte
	ReceiveDate time.Time
	SourcePoint *gopacket.Endpoint
	TargetPoint *gopacket.Endpoint
	SourceHost  *gopacket.Endpoint
	TargetHost  *gopacket.Endpoint
	TCP         *layers.TCP
}

//Decode 不同协议解码器接口
type Decode interface {
	Decode(*SourceData)
}

//FindDecode 通过协议查找解码器
func FindDecode(protol string) Decode {
	switch protol {
	case "http":
		return &HTTPDecode{}
	default:
		return nil
	}
}
