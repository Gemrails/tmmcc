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
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/quipo/statsd"
)

//TIMEBUCKETS Internal tuning
const TIMEBUCKETS = 10000

//Maximume max 10
const Maximume = 10

//MapKey MapKey
type MapKey string
type httpMetricStore struct {
	methodRequestSize  map[string]uint64
	unusualRequestSize map[string]uint64
	requestTimes       [TIMEBUCKETS]uint64
	//每次发出消息后清理
	PathCache map[string]*cache
	//每次发出消息后清理
	IndependentIP        map[string]*cache
	ServiceID            string
	Port                 string
	ctx                  context.Context
	cancel               context.CancelFunc
	lock                 sync.Mutex
	monitorMessageManage *MonitorMessageManage
	statsdclient         *statsd.StatsdClient
}

func (h *httpMetricStore) show() {
	h.lock.Lock()
	defer h.lock.Unlock()
	fmt.Println("-------methodRequest--------------")
	for k, v := range h.methodRequestSize {
		fmt.Printf("Path:%s   Count %d \n", k, v)
	}
	fmt.Println("-------unusualRequestSize---------")
	for k, v := range h.unusualRequestSize {
		fmt.Printf("Path:%s   Count %d \n", k, v)
	}
	fmt.Println("-------requestTimes---------------")
	min, avg, max := calculate(&h.requestTimes)
	fmt.Printf("Min %f  Avg %f  Max %f \n", min, avg, max)
	fmt.Println("-------cacheRequest---------------")
	for k, v := range h.PathCache {
		min, avg, max := calculate(&v.ResTime)
		fmt.Printf("Path:%s   Count %d UnusualCount %d timemin %f timeavg %f timemax %f \n", k, v.Count, v.UnusualCount, min, avg, max)
	}
	fmt.Println("-------IndependentIP---------")
	for k, v := range h.IndependentIP {
		fmt.Printf("IndependentIP:%s   Count %d \n", k, v)
	}
}

//sendmessage send message to eventlog monitor message chan
func (h *httpMetricStore) sendmessage() {
	h.lock.Lock()
	defer h.lock.Unlock()
	var caches = new(MonitorMessageList)
	for _, v := range h.PathCache {
		_, avg, max := calculate(&v.ResTime)
		mm := MonitorMessage{
			ServiceID:      h.ServiceID,
			Port:           h.Port,
			MessageType:    "http",
			Key:            v.Key,
			Count:          v.Count,
			AbnormalCount:  v.UnusualCount,
			AverageTime:    Round(avg, 2),
			MaxTime:        Round(max, 2),
			CumulativeTime: Round(avg*float64(v.Count), 2),
		}
		caches.Add(&mm)
	}
	sort.Sort(caches)
	if caches.Len() > 20 {
		h.monitorMessageManage.Send(caches.Pop(20))
		return
	}
	h.monitorMessageManage.Send(caches)
}

//sendstatsd send metric to statsd
func (h *httpMetricStore) sendstatsd() {
	h.lock.Lock()
	defer h.lock.Unlock()
	var total int64
	for k, v := range h.methodRequestSize {
		h.statsdclient.Incr("request."+k, int64(v))
		total += int64(v)
		h.methodRequestSize[k] = 0
	}
	h.statsdclient.Incr("request.total", int64(total))
	var errtotal int64
	for k, v := range h.unusualRequestSize {
		h.statsdclient.Incr("request.unusual."+k, int64(v))
		errtotal += int64(v)
		h.unusualRequestSize[k] = 0
	}
	h.statsdclient.Incr("request.unusual.total", int64(errtotal))
	min, avg, max := calculate(&h.requestTimes)
	h.statsdclient.FGauge("requesttime.min", min)
	h.statsdclient.FGauge("requesttime.avg", avg)
	h.statsdclient.FGauge("requesttime.max", max)
	h.statsdclient.Gauge("request.client", int64(len(h.IndependentIP)))
}

func (h *httpMetricStore) clear() {
	var clearKey []string
	for k, v := range h.PathCache {
		if v.updateTime.Add(5 * time.Minute).Before(time.Now()) {
			clearKey = append(clearKey, k)
		}
	}
	for _, key := range clearKey {
		delete(h.PathCache, key)
	}
	clearKey = clearKey[:0]
	for k, v := range h.IndependentIP {
		if v.updateTime.Add(5 * time.Minute).Before(time.Now()) {
			clearKey = append(clearKey, k)
		}
	}
	for _, key := range clearKey {
		delete(h.IndependentIP, key)
	}

}

//Input 数据输入
func (h *httpMetricStore) Input(message interface{}) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if httpms, ok := message.(*HTTPMessage); ok {
		//request method
		h.methodRequestSize[httpms.Method]++
		//request unusual
		if httpms.StatusCode >= 400 && httpms.StatusCode < 500 {
			h.unusualRequestSize["4xx"]++
		}
		if httpms.StatusCode >= 500 {
			h.unusualRequestSize["5xx"]++
		}
		//requestTimes
		randn := rand.Intn(TIMEBUCKETS)
		h.requestTimes[randn] = uint64(httpms.TimeConsum)
		//cache
		if c, ok := h.PathCache[httpms.URI]; ok {
			c.Count++
			if httpms.StatusCode >= 400 {
				c.UnusualCount++
			}
			c.ResTime[randn] = uint64(httpms.TimeConsum)
			c.updateTime = time.Now()
		} else {
			c := &cache{
				Key: httpms.URI,
			}
			c.Count++
			if httpms.StatusCode >= 400 {
				c.UnusualCount++
			}
			c.ResTime[randn] = uint64(httpms.TimeConsum)
			c.updateTime = time.Now()
			h.PathCache[httpms.URI] = c
		}
		//remote addr
		if c, ok := h.IndependentIP[httpms.RemoteAddr]; ok {
			c.Count++
			c.updateTime = time.Now()
		} else {
			c := &cache{
				Key: httpms.RemoteAddr,
			}
			c.Count++
			c.updateTime = time.Now()
			h.IndependentIP[httpms.RemoteAddr] = c
		}
	}
}

//Start 启动
func (h *httpMetricStore) Start() {
	tickMessage := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-h.ctx.Done():
			tickMessage.Stop()
			return
		case <-tickMessage.C:
			h.sendmessage()
			h.sendstatsd()
			h.clear()
		}
	}
}

//Stop 停止
func (h *httpMetricStore) Stop() {
	h.cancel()
}

//HTTPMessage http protocol zeromq message
type HTTPMessage struct {
	Method        string `json:"method"`
	URI           string `json:"uri"`
	StatusCode    int    `json:"statusCode"`
	ContentLength int    `json:"contentLength"`
	TimeConsum    int64  `json:"timeConsum"`
	RemoteAddr    string
}

//CreateHTTPMessage 通过response构造message
func CreateHTTPMessage(rs *http.Response) *HTTPMessage {
	m := &HTTPMessage{
		Method:     rs.Request.Method,
		StatusCode: rs.StatusCode,
		RemoteAddr: rs.Request.RemoteAddr,
	}
	in := strings.Index(rs.Request.RequestURI, "?")
	if in > -1 {
		m.URI = rs.Request.RequestURI[:in]
	} else {
		m.URI = rs.Request.RequestURI
	}
	var ReqTime, ResTime time.Time
	if t, ok := rs.Request.Context().Value(MapKey("ReqTime")).(time.Time); ok {
		ReqTime = t
	}
	if t, ok := rs.Request.Context().Value(MapKey("ResTime")).(time.Time); ok {
		ResTime = t
	}
	if !ReqTime.IsZero() && !ResTime.IsZero() {
		m.TimeConsum = ResTime.Sub(ReqTime).Nanoseconds()
	}
	return m
}

//Round Round
func Round(f float64, n int) float64 {
	pow10n := math.Pow10(n)
	return math.Trunc((f+0.5/pow10n)*pow10n) / pow10n
}
