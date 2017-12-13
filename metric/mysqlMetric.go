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
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/quipo/statsd"
)

type mysqlMetricStore struct {
	sqlRequestSize map[string]uint64
	requestTimes   [TIMEBUCKETS]uint64
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

//sendmessage send message to eventlog monitor message chan
func (h *mysqlMetricStore) sendmessage() {
	h.lock.Lock()
	defer h.lock.Unlock()
	var caches = new(MonitorMessageList)
	for _, v := range h.PathCache {
		_, avg, max := calculate(&v.ResTime)
		mm := MonitorMessage{
			ServiceID:      h.ServiceID,
			Port:           h.Port,
			MessageType:    "mysql",
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
func (h *mysqlMetricStore) sendstatsd() {
	h.lock.Lock()
	defer h.lock.Unlock()
	var total int64
	for k, v := range h.sqlRequestSize {
		h.statsdclient.Incr("request."+k, int64(v))
		total += int64(v)
		h.sqlRequestSize[k] = 0
	}
	h.statsdclient.Incr("request.total", int64(total))
	min, avg, max := calculate(&h.requestTimes)
	h.statsdclient.FGauge("requesttime.min", min)
	h.statsdclient.FGauge("requesttime.avg", avg)
	h.statsdclient.FGauge("requesttime.max", max)
	h.statsdclient.Gauge("request.client", int64(len(h.IndependentIP)))
}

func (h *mysqlMetricStore) clear() {
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
func (h *mysqlMetricStore) Input(message interface{}) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if mm, ok := message.(*MysqlMessage); ok {
		randn := rand.Intn(TIMEBUCKETS)
		h.requestTimes[randn] = mm.Reqtime
		h.sqlRequestSize[mm.Code]++
		//cache
		if c, ok := h.PathCache[mm.SQL]; ok {
			c.Count++
			if mm.Code != "Success" {
				c.UnusualCount++
			}
			c.ResTime[randn] = mm.Reqtime
			c.ResLength += mm.ContentLength
			c.updateTime = time.Now()
		} else {
			c := &cache{
				Key: mm.SQL,
			}
			c.Count++
			if mm.Code != "Success" {
				c.UnusualCount++
			}
			c.ResTime[randn] = mm.Reqtime
			c.updateTime = time.Now()
			h.PathCache[mm.SQL] = c
		}
		//remote addr
		if c, ok := h.IndependentIP[mm.RemoteAddr]; ok {
			c.Count++
			c.updateTime = time.Now()
		} else {
			c := &cache{
				Key: mm.RemoteAddr,
			}
			c.Count++
			c.updateTime = time.Now()
			h.IndependentIP[mm.RemoteAddr] = c
		}
	}
}

//Start 启动
func (h *mysqlMetricStore) Start() {
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
func (h *mysqlMetricStore) Stop() {
	h.cancel()
}

//MysqlMessage mysql protocol message
type MysqlMessage struct {
	Code          string `json:"code"`
	SQL           string `json:"sql"`
	ContentLength uint64 `json:"contentLength"`
	Reqtime       uint64 `json:"reqtime"`
	RemoteAddr    string
}
