package net

import (
	"net/http"
	"time"

	"github.com/prometheus/common/log"
)

//HTTPMessage http protocol zeromq message
type HTTPMessage struct {
	Method        string    `json:"method"`
	URI           string    `json:"uri"`
	Protocol      string    `json:"protocol"`
	UserAgent     string    `json:"userAgent"`
	Referer       string    `json:"referer"`
	Cookie        string    `json:"cookie"`
	ReqTime       time.Time `json:"reqTime"`
	Server        string    `json:"server"`
	StatusCode    int       `json:"statusCode"`
	ContentLength int       `json:"contentLength"`
	ResTime       time.Time `json:"resTime"`
	SourceIP      string    `json:"sourceIP"`
}

//CreateHTTPMessage 通过response构造message
func CreateHTTPMessage(*http.Response) *HTTPMessage {
	m := &HTTPMessage{}

	return m
}

//MessageManager message消化接口
type MessageManager interface {
	SendMessage(interface{})
}

//ZMQMessageManager zmq消化message
type ZMQMessageManager struct {
	ZMQURL string
	send   chan interface{}
	//client zmq.Sender
}

var zmqmanager MessageManager

//CreateZMQMessageManager 创建zmq 消化器
func CreateZMQMessageManager(url string, close chan struct{}) {
	if zmqmanager == nil {
		manager := &ZMQMessageManager{
			ZMQURL: url,
			send:   make(chan interface{}, 10),
		}
		go manager.msend(close)
		zmqmanager = manager
	}
}

//SendMessage 发送message
func (z *ZMQMessageManager) SendMessage(m interface{}) {
	z.send <- m
}
func (z *ZMQMessageManager) msend(close chan struct{}) {
	for {
		select {
		case <-close:
			return
		case message := <-z.send:
			log.Infoln(message)
			//z.client.Send(message)
		}
	}
}
