package net

import (
	"bufio"
	"bytes"
	"net/http"
	"time"

	"sync"

	conv "github.com/cstockton/go-conv"
	"github.com/prometheus/common/log"
	"golang.org/x/net/context"
)

//HTTPDecode http解码
type HTTPDecode struct {
}
type mapkey string

//Decode 解码
func (h *HTTPDecode) Decode(data *SourceData) {
	bufer := bytes.NewBuffer(data.Source)
	if string(data.Source[0:4]) == "HTTP" { //Response
		if data.TCP != nil {
			key := conv.String(data.TCP.Seq) + data.TargetHost.String() + ":" + data.TargetPoint.String()
			re := httpmanager.GetRequest(key)
			if re == nil {
				for i := 5; i > 0; i-- {
					time.Sleep(2 * time.Second)
					re = httpmanager.GetRequest(key)
					if re != nil {
						break
					}
				}
				if re == nil {
					log.Errorln("don't find request with response seq after 10s,response loss.")
					return
				}
			}
			response, err := http.ReadResponse(bufio.NewReader(bufer), re)
			if err != nil {
				log.With("error", err.Error()).Errorln("Decode the data to http response error.")
			}
			if response != nil {
				log.Infof("To Host %s Port %s Time: %s ", data.TargetHost.String(), data.TargetPoint.String(), conv.String(data.ReceiveDate))
				log.Infof("Response ACK %b:%d  SEQ %d", data.TCP.ACK, data.TCP.Ack, data.TCP.Seq)
				httpmanager.ResponseSetChan <- response
			}
		}
	} else { //Request
		log.Infof("From Host %s Port %s  Time: %s ", data.SourceHost.String(), data.SourcePoint.String(), conv.String(data.ReceiveDate))
		request, err := http.ReadRequest(bufio.NewReader(bufer))
		if err != nil {
			log.With("error", err.Error()).Errorln("Decode the data to http request error.")
			if data.TCP != nil {
				log.Infoln(string(data.TCP.BaseLayer.Contents))
			}
			log.Infoln(string(data.Source))
		}
		if request != nil && data.TCP != nil {
			log.Infof("From Request Path %s ", request.RequestURI)
			log.Infof("Request ACK %b:%d  SEQ %d ", data.TCP.ACK, data.TCP.Ack, data.TCP.Seq)
			key := conv.String(data.TCP.Ack) + data.SourceHost.String() + ":" + data.SourcePoint.String()
			request := request.WithContext(context.WithValue(context.Background(), mapkey("key"), key))
			httpmanager.RequestSetChan <- request
		}
	}
}

//HTTPManager 监控信息存储
type HTTPManager struct {
	requests                   map[string]*http.Request
	responses                  map[string]*http.Response
	RequestSetChan             chan *http.Request
	ResponseSetChan            chan *http.Response
	RequestsLock, ResponseLock sync.Mutex
}

var httpmanager *HTTPManager

//CreateHTTPManager 创建httpmanager
func CreateHTTPManager(close chan struct{}) {
	if httpmanager == nil {
		httpmanager = &HTTPManager{
			requests:        make(map[string]*http.Request, 10),
			responses:       make(map[string]*http.Response, 10),
			RequestSetChan:  make(chan *http.Request, 10),
			ResponseSetChan: make(chan *http.Response, 10),
		}
		go httpmanager.readRequestChan(close)
		go httpmanager.readResponseChan(close)
	}
}

//Close 关闭
func (m *HTTPManager) Close() {
	if m.RequestSetChan != nil {
		close(m.RequestSetChan)
	}
	if m.ResponseSetChan != nil {
		close(m.ResponseSetChan)
	}
}

func (m *HTTPManager) readRequestChan(close chan struct{}) {
	for {
		select {
		case <-close:
			return
		case request := <-m.RequestSetChan:
			m.RequestsLock.Lock()
			key := request.Context().Value(mapkey("key")).(string)
			m.requests[key] = request
			log.Infof("Request number:%d", len(m.requests))
			m.RequestsLock.Unlock()
		}
	}
}
func (m *HTTPManager) readResponseChan(close chan struct{}) {
	for {
		select {
		case <-close:
			return
		case response := <-m.ResponseSetChan:
			m.RequestsLock.Lock()
			key := response.Request.Context().Value(mapkey("key")).(string)
			//m.responses[key] = response
			delete(m.requests, key)
			//log.Infof("Response number:%d", len(m.responses))
			m.RequestsLock.Unlock()
		}
	}
}

//GetRequest 获取request
func (m *HTTPManager) GetRequest(key string) *http.Request {
	m.RequestsLock.Lock()
	defer m.RequestsLock.Unlock()
	return m.requests[key]
}
