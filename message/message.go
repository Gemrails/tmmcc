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

package message

import (
	"fmt"
	"net"
	"time"

	"encoding/json"

	"github.com/prometheus/common/log"
)

//Message 消息
type Message interface {
	String() string
	Byte() []byte
}

//HTTPMessage http protocol zeromq message
type HTTPMessage struct {
	ServiceID     string    `json:"serviceID"`
	Method        string    `json:"method"`
	URI           string    `json:"uri"`
	Protocol      string    `json:"protocol"`
	UserAgent     string    `json:"userAgent"`
	ReqTime       time.Time `json:"reqTime"`
	Server        string    `json:"server"`
	StatusCode    int       `json:"statusCode"`
	ContentLength int       `json:"contentLength"`
	ResTime       time.Time `json:"resTime"`
	RemoteAddr    string    `json:"remoteAddr"`
	TimeConsum    float64   `json:"timeConsum"`
}

//String 返回json格式
func (m *HTTPMessage) String() string {
	d, err := json.Marshal(m)
	if err != nil {
		log.With("error", err.Error()).Error("HTTP message to json error.")
		return ""
	}
	return string(d)
}

//Byte Byte
func (m *HTTPMessage) Byte() []byte {
	d, err := json.Marshal(m)
	if err != nil {
		log.With("error", err.Error()).Error("HTTP message to json error.")
		return nil
	}
	return d
}

//MessageManager message消化接口
type MessageManager interface {
	SendMessage(Message)
}

//GetMessageManager 获取message消化器
func GetMessageManager(host string, port int, sendCount int, close chan struct{}, managerType string) (MessageManager, error) {
	switch managerType {
	case "http":
		m, err := CreateMessageManager(host, port, sendCount, close)
		if err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, fmt.Errorf("no message type manage support")
	}
}

//UDPMessageManager message push with udp
type UDPMessageManager struct {
	Host   string
	Port   int
	client net.Conn
}

var udpmanager MessageManager

//CreateMessageManager 创建zmq 消化器
func CreateMessageManager(host string, port int, sendCount int, close chan struct{}) (MessageManager, error) {
	if sendCount == 0 {
		sendCount = 1
	}
	if udpmanager == nil {
		manager := &UDPMessageManager{
			Host: host,
			Port: port,
		}
		dip := net.ParseIP(host)
		srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
		dstAddr := &net.UDPAddr{IP: dip, Port: port}
		conn, err := net.DialUDP("udp", srcAddr, dstAddr)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		manager.client = conn
		udpmanager = manager
	}
	return udpmanager, nil
}

//SendMessage 发送message
func (z *UDPMessageManager) SendMessage(m Message) {
	z.client.Write(m.Byte())
}
