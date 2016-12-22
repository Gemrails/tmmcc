package net

import (
	"bufio"
	"bytes"
	"net/http"
	"strings"
	"tcm/config"
	"time"

	"sync"

	"os"

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
	if data.TCP == nil {
		log.Errorln("TCP is nil, so it may be is not http")
		return
	}
	if data.TCP.SrcPort.String() == "" {
		log.Errorln("TCP SrcPort is empty, so it may be is not http")
		return
	}
	Port := os.Getenv("PORT")
	if strings.HasPrefix(data.TCP.SrcPort.String(), Port+"(") { //Response,通过源端口判断
		//抛弃数据包，只分析报文头部包
		if string(data.Source[0:4]) == "HTTP" {
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
			re = re.WithContext(context.WithValue(re.Context(), mapkey("ResTime"), conv.String(data.ReceiveDate)))
			response, err := http.ReadResponse(bufio.NewReader(bufer), re)
			if err != nil {
				log.With("error", err.Error()).Errorln("Decode the data to http response error.")
			}
			if response != nil {
				// log.Infof("To Host %s Port %s Time: %s ", data.TargetHost.String(), data.TargetPoint.String(), conv.String(data.ReceiveDate))
				// log.Infof("Response ACK %b:%d  SEQ %d  Content-length:%s", data.TCP.ACK, data.TCP.Ack, data.TCP.Seq, response.Header.Get("Content-Length"))
				httpmanager.ResponseSetChan <- response
			}
		} else {
			log.Infoln("Discarded Response body package ")
		}
	} else { //Request
		log.Infof("From Host %s Port %s  Time: %s ", data.SourceHost.String(), data.SourcePoint.String(), conv.String(data.ReceiveDate))
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
			request = request.WithContext(context.WithValue(request.Context(), mapkey("ReqTime"), conv.String(data.ReceiveDate)))
			httpmanager.RequestSetChan <- request
		}
	}
}

//HTTPManager 监控信息存储
type HTTPManager struct {
	requests                   map[string]http.Request
	RequestSetChan             chan *http.Request
	ResponseSetChan            chan *http.Response
	RequestsLock, ResponseLock sync.Mutex
	messageManager             MessageManager
}

var httpmanager *HTTPManager

//CreateHTTPManager 创建httpmanager
func CreateHTTPManager(option *config.Option) error {
	if httpmanager == nil {
		httpmanager = &HTTPManager{
			requests:        make(map[string]http.Request, 10),
			RequestSetChan:  make(chan *http.Request, 10),
			ResponseSetChan: make(chan *http.Response, 10),
			messageManager:  GetMessageManager(option.ZMQURL, option.SendCount, option.Close, "http"),
		}
		go httpmanager.readRequestChan(option.Close)
		go httpmanager.readResponseChan(option.Close)
	}
	return nil
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
			log.Infoln("stop read request chan")
			return
		case request := <-m.RequestSetChan:
			m.RequestsLock.Lock()
			key := request.Context().Value(mapkey("key")).(string)
			m.requests[key] = *request
			//log.Infof("Request number:%d", len(m.requests))
			m.RequestsLock.Unlock()
		}
	}
}
func (m *HTTPManager) readResponseChan(close chan struct{}) {
	for {
		select {
		case <-close:
			log.Infoln("stop read response chan")
			return
		case response := <-m.ResponseSetChan:
			m.RequestsLock.Lock()
			key := response.Request.Context().Value(mapkey("key")).(string)
			delete(m.requests, key)
			message := CreateHTTPMessage(response)
			m.messageManager.SendMessage(message)
			m.RequestsLock.Unlock()
		}
	}
}

//GetRequest 获取request
func (m *HTTPManager) GetRequest(key string) *http.Request {
	m.RequestsLock.Lock()
	defer m.RequestsLock.Unlock()
	re, ok := m.requests[key]
	if ok {
		return &re
	}
	return nil
}
