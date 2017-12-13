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

package metric

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/prometheus/common/log"

	"github.com/quipo/statsd"
)

//Store data store
type Store interface {
	Input(interface{})
	Start()
	Stop()
}

//NewMetric new metric store
func NewMetric(protocol, messageServer, statsdserver string, messagePort int) Store {
	mmm, err := CreateMonitorMessageManage(messageServer, messagePort)
	if err != nil {
		return nil
	}
	os.Setenv("SERVICE_ID", "test")
	os.Setenv("PORT", "5000")
	statsdclient := CreateStatsdClient(statsdserver, fmt.Sprintf("%s.%s.", os.Getenv("SERVICE_ID"), os.Getenv("PORT")))
	if statsdclient == nil {
		return nil
	}
	switch protocol {
	case "http":
		ctx, cancel := context.WithCancel(context.Background())
		return &httpMetricStore{
			methodRequestSize:    make(map[string]uint64),
			unusualRequestSize:   make(map[string]uint64),
			PathCache:            make(map[string]*cache),
			IndependentIP:        make(map[string]*cache),
			ServiceID:            os.Getenv("SERVICE_ID"),
			Port:                 os.Getenv("PORT"),
			cancel:               cancel,
			ctx:                  ctx,
			monitorMessageManage: mmm,
			statsdclient:         statsdclient,
		}
	case "mysql":
		ctx, cancel := context.WithCancel(context.Background())
		return &mysqlMetricStore{
			sqlRequestSize:       make(map[string]uint64),
			PathCache:            make(map[string]*cache),
			IndependentIP:        make(map[string]*cache),
			ServiceID:            os.Getenv("SERVICE_ID"),
			Port:                 os.Getenv("PORT"),
			cancel:               cancel,
			ctx:                  ctx,
			monitorMessageManage: mmm,
			statsdclient:         statsdclient,
		}
	default:
		return nil
	}
}

//MonitorMessage 性能监控消息系统模型
type MonitorMessage struct {
	ServiceID   string
	Port        string
	MessageType string //mysql，http ...
	Key         string
	//总时间
	CumulativeTime float64
	AverageTime    float64
	MaxTime        float64
	Count          uint64
	//异常请求次数
	AbnormalCount uint64
}

//MonitorMessageList 消息列表
type MonitorMessageList []MonitorMessage

//Add 添加
func (m *MonitorMessageList) Add(mm *MonitorMessage) {
	*m = append(*m, *mm)
}

// Len 为集合内元素的总数
func (m *MonitorMessageList) Len() int {
	return len(*m)
}

//Less 如果index为i的元素小于index为j的元素，则返回true，否则返回false
func (m *MonitorMessageList) Less(i, j int) bool {
	return (*m)[i].CumulativeTime < (*m)[j].CumulativeTime
}

// Swap 交换索引为 i 和 j 的元素
func (m *MonitorMessageList) Swap(i, j int) {
	tmp := (*m)[i]
	(*m)[i] = (*m)[j]
	(*m)[j] = tmp
}

//Pop Pop
func (m *MonitorMessageList) Pop(i int) *MonitorMessageList {
	cache := (*m)[:i]
	return &cache
}

//String json string
func (m *MonitorMessageList) String() string {
	body, _ := json.Marshal(m)
	return string(body)
}

//MonitorMessageManage monitor message manage
type MonitorMessageManage struct {
	ServerAddr string
	Port       int
	client     net.Conn
}

//CreateMonitorMessageManage create
func CreateMonitorMessageManage(server string, port int) (*MonitorMessageManage, error) {
	manager := &MonitorMessageManage{
		ServerAddr: server,
		Port:       port,
	}
	dip := net.ParseIP(server)
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{IP: dip, Port: port}
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		log.Error("create monitor message send client error,", err)
		return nil, err
	}
	manager.client = conn
	return manager, nil
}

//Send message send
func (m *MonitorMessageManage) Send(mml *MonitorMessageList) {
	if mml != nil && mml.Len() > 0 {
		m.client.Write([]byte(mml.String()))
	}
}

//CreateStatsdClient create client
func CreateStatsdClient(server, prefix string) *statsd.StatsdClient {
	statsdclient := statsd.NewStatsdClient(server, prefix)
	err := statsdclient.CreateSocket()
	if nil != err {
		log.Error("create statsd client error,", err)
		return nil
	}
	return statsdclient
}

type cache struct {
	Key          string
	Count        uint64
	UnusualCount uint64
	ResTime      [TIMEBUCKETS]uint64
	updateTime   time.Time
	ResLength    uint64
}
