package net

import (
	"net/http"

	"encoding/json"

	conv "github.com/cstockton/go-conv"
	"github.com/prometheus/common/log"
)

//Message 消息
type Message interface {
	GetJSON() string
}

//HTTPMessage http protocol zeromq message
type HTTPMessage struct {
	Method        string         `json:"method"`
	URI           string         `json:"uri"`
	Protocol      string         `json:"protocol"`
	UserAgent     string         `json:"userAgent"`
	Referer       string         `json:"referer"`
	Cookie        []*http.Cookie `json:"cookie"`
	ReqTime       string         `json:"reqTime"`
	Server        string         `json:"server"`
	StatusCode    int            `json:"statusCode"`
	ContentLength int            `json:"contentLength"`
	ResTime       string         `json:"resTime"`
	RemoteAddr    string         `json:"remoteAddr"`
}

//GetJSON 返回json格式
func (m *HTTPMessage) GetJSON() string {
	d, err := json.Marshal(m)
	if err != nil {
		log.With("error", err.Error()).Error("HTTP message to json error.")
		return ""
	}
	return string(d)
}

//CreateHTTPMessage 通过response构造message
func CreateHTTPMessage(rs *http.Response) *HTTPMessage {

	m := &HTTPMessage{
		Method:        rs.Request.Method,
		URI:           rs.Request.RequestURI,
		Protocol:      rs.Request.Proto,
		UserAgent:     rs.Request.UserAgent(),
		Referer:       rs.Request.Referer(),
		Cookie:        rs.Request.Cookies(),
		ReqTime:       rs.Request.Context().Value(mapkey("ReqTime")).(string),
		Server:        rs.Header.Get("Server"),
		StatusCode:    rs.StatusCode,
		ContentLength: conv.Int(rs.Header.Get("Content-Length")),
		ResTime:       rs.Request.Context().Value(mapkey("ResTime")).(string),
		RemoteAddr:    rs.Request.RemoteAddr,
	}
	rs.Request.Close = true
	rs.Close = true
	return m
}

//MessageManager message消化接口
type MessageManager interface {
	SendMessage(Message)
}

//GetMessageManager 获取message消化器
func GetMessageManager(zmqurl string, sendCount int, close chan struct{}, managerType string) MessageManager {
	switch managerType {
	case "http":
		return CreateZMQMessageManager(zmqurl, sendCount, close)
	default:
		return nil
	}
}

//ZMQMessageManager zmq消化message
type ZMQMessageManager struct {
	ZMQURL string
	send   chan Message
	//client zmq.Sender
}

var zmqmanager MessageManager

//CreateZMQMessageManager 创建zmq 消化器
func CreateZMQMessageManager(url string, sendCount int, close chan struct{}) MessageManager {
	if sendCount == 0 {
		sendCount = 1
	}
	if zmqmanager == nil {
		manager := &ZMQMessageManager{
			ZMQURL: url,
			send:   make(chan Message, 10),
		}
		for i := sendCount; i > 0; i-- {
			go manager.msend(close)
		}
		zmqmanager = manager
	}
	return zmqmanager
}

//SendMessage 发送message
func (z *ZMQMessageManager) SendMessage(m Message) {
	z.send <- m
}
func (z *ZMQMessageManager) msend(close chan struct{}) {
	log.Infoln("Start read message and sent to zmq")
	for {
		select {
		case <-close:
			log.Infoln("stop read and sent message")
			return
		case message := <-z.send:
			body := message.GetJSON()
			if body != "" {
				log.Info(body)
			}
			//z.client.Send(message)
		}
	}
}
