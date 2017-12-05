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
	"bufio"
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"tcm/config"
	"time"

	"sync"

	conv "github.com/cstockton/go-conv"
	cache "github.com/patrickmn/go-cache"
	"github.com/prometheus/common/log"
)

//HTTPDecode http解码
type HTTPDecode struct {
}
type mapkey string

//Decode 解码
func (h *HTTPDecode) Decode(data *SourceData) {
	bufer := bytes.NewBuffer(data.Source)
	//fmt.Println(string(data.Source))
	if data.TCP == nil {
		log.Errorln("TCP is nil, so it may be is not http")
		return
	}
	if data.TCP.SrcPort.String() == "" {
		log.Errorln("TCP SrcPort is empty, so it may be is not http")
		return
	}
	Port := os.Getenv("PORT")
	if Port == "" {
		Port = "5000"
	}
	if strings.HasPrefix(data.TCP.SrcPort.String(), Port+"(") { //Response,通过源端口判断
		//抛弃数据包，只分析报文头部包
		if string(data.Source[0:4]) == "HTTP" {
			var rm ResponseMessage
			rm.RequestKey = conv.String(data.TCP.Seq) + data.TargetHost.String() + ":" + data.TargetPoint.String()
			rm.ReceiveTime = data.ReceiveDate
			response, err := http.ReadResponse(bufio.NewReader(bufer), nil)
			if err != nil {
				log.With("error", err.Error()).Errorln("Decode the data to http response error.")
			}
			if response != nil {
				// log.Infof("To Host %s Port %s Time: %s ", data.TargetHost.String(), data.TargetPoint.String(), conv.String(data.ReceiveDate))
				// log.Infof("Response ACK %b:%d  SEQ %d  Content-length:%s", data.TCP.ACK, data.TCP.Ack, data.TCP.Seq, response.Header.Get("Content-Length"))
				rm.Response = response
				httpmanager.MessageChan <- rm
			}
		} else {
			log.Warnln("Discarded Response body package ", data.Source[0:4])
		}
	} else { //Request
		//log.Infof("From Host %s Port %s  Time: %s ", data.SourceHost.String(), data.SourcePoint.String(), conv.String(data.ReceiveDate))
		request, err := http.ReadRequest(bufio.NewReader(bufer))
		if err != nil {
			log.With("error", err.Error()).Errorln("Decode the data to http request error.")
		}
		if request != nil && data.TCP != nil {
			// log.Infof("From Request Path %s ", request.RequestURI)
			// log.Infof("Request ACK %b:%d  SEQ %d ", data.TCP.ACK, data.TCP.Ack, data.TCP.Seq)
			request.RemoteAddr = data.SourceHost.String()
			key := conv.String(data.TCP.Ack) + data.SourceHost.String() + ":" + data.SourcePoint.String()
			request = request.WithContext(context.WithValue(context.Background(), mapkey("key"), key))
			request = request.WithContext(context.WithValue(request.Context(), mapkey("ReqTime"), data.ReceiveDate))
			httpmanager.MessageChan <- request
		}
	}
}

//HTTPManager 监控信息存储
type HTTPManager struct {
	cache                      *cache.Cache
	MessageChan                chan interface{}
	RequestsLock, ResponseLock sync.Mutex
	messageManager             MessageManager
}

//ResponseMessage response message
type ResponseMessage struct {
	Response    *http.Response
	RequestKey  string
	ReceiveTime time.Time
}

var httpmanager *HTTPManager

//CreateHTTPManager 创建httpmanager
func CreateHTTPManager(option *config.Option) error {
	if httpmanager == nil {
		m, err := GetMessageManager(option.UDPIP, option.UDPPort, option.SendCount, option.Close, "http")
		if err != nil {
			return err
		}
		httpmanager = &HTTPManager{
			cache:          cache.New(10*time.Second, 1*time.Minute),
			MessageChan:    make(chan interface{}, 100),
			messageManager: m,
		}
		go httpmanager.handleMessageChan(option.Close)
	}
	return nil
}

//Close 关闭
func (m *HTTPManager) Close() {
	if m.MessageChan != nil {
		close(m.MessageChan)
	}
}
func (m *HTTPManager) handleMessageChan(close chan struct{}) {
	for {
		select {
		case <-close:
			log.Infoln("stop read request message chan")
			return
		case message := <-m.MessageChan:
			switch message.(type) {
			case *http.Request:
				request := message.(*http.Request)
				key := request.Context().Value(mapkey("key")).(string)
				m.cache.Set(key, request, cache.DefaultExpiration)
				//log.Infof("Request number:%d", len(m.requests))
			case ResponseMessage:
				response := message.(ResponseMessage)
				key := response.RequestKey
				if re, ok := m.cache.Get(key); ok {
					if r, ok := re.(*http.Request); ok {
						r = r.WithContext(context.WithValue(r.Context(), mapkey("ResTime"), response.ReceiveTime))
						response.Response.Request = r
						m.cache.Delete(key)
						info := CreateHTTPMessage(response.Response)
						m.messageManager.SendMessage(info)
					}
				} else {
					log.Warnf("request key %s not found", key)
					continue
				}

			}
		}
	}
}
