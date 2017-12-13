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
	"fmt"
	"math/rand"
	"os"
	"strings"
	"tcm/config"
	"tcm/metric"
	"time"

	"github.com/prometheus/common/log"
)

//MysqlDecode http解码
type MysqlDecode struct {
	qbuf             map[string]*queryData
	chmap            map[string]*source
	format           []interface{}
	mysqlMetricStore metric.Store
}

//CreateMysqlDecode CreateMysqlDecode
func CreateMysqlDecode(option *config.Option) *MysqlDecode {
	ms := metric.NewMetric("mysql", option.UDPIP, option.StatsdServer, option.UDPPort)
	if ms == nil {
		log.Errorf("create metric store error")
		return nil
	}
	go ms.Start()
	m := MysqlDecode{
		qbuf:             make(map[string]*queryData),
		chmap:            make(map[string]*source),
		mysqlMetricStore: ms,
	}
	m.parseFormat("#s:#q")
	rand.Seed(time.Now().UnixNano())
	return &m
}

//Decode 解码
func (h *MysqlDecode) Decode(data *SourceData) {
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
	// This is either an inbound or outbound packet. Determine by seeing which
	// end contains our port. Either way, we want to put this on the channel of
	// the remote end.
	var src string
	var request bool
	if strings.HasPrefix(data.TCP.SrcPort.String(), "3306") {
		src = data.TargetHost.String() + ":" + data.TargetPoint.String()
		//log.Printf("response to %s", src)
	} else {
		src = data.SourceHost.String() + ":" + data.SourcePoint.String()
		request = true
		//log.Printf("request from %s", src)
	}

	// Get the data structure for this source, then do something.
	rs, ok := h.chmap[src]
	if !ok {
		rs = &source{src: src, srcip: data.SourceHost.String(), synced: false}
		stats.streams++
		h.chmap[src] = rs
	}
	//fmt.Println(data.Source)
	// Now with a source, process the packet.
	h.processPacket(rs, request, data.Source)

}

const (
	TOKEN_WORD       = 0
	TOKEN_QUOTE      = 1
	TOKEN_NUMBER     = 2
	TOKEN_WHITESPACE = 3
	TOKEN_OTHER      = 4

	// Internal tuning
	TIME_BUCKETS = 10000

	// ANSI colors
	COLOR_RED     = "\x1b[31m"
	COLOR_GREEN   = "\x1b[32m"
	COLOR_YELLOW  = "\x1b[33m"
	COLOR_CYAN    = "\x1b[36m"
	COLOR_WHITE   = "\x1b[37m"
	COLOR_DEFAULT = "\x1b[39m"

	// MySQL packet types
	COM_QUERY = 3

	// These are used for formatting outputs
	F_NONE = iota
	F_QUERY
	F_ROUTE
	F_SOURCE
	F_SOURCEIP
)

type packet struct {
	request bool // request or response
	data    []byte
}

type sortable struct {
	value float64
	line  string
}
type sortableSlice []sortable

type source struct {
	src       string
	srcip     string
	synced    bool
	reqbuffer []byte
	resbuffer []byte
	reqSent   *time.Time
	reqTimes  [TIME_BUCKETS]uint64
	qbytes    uint64
	qdata     *queryData
	qtext     string
}

type queryData struct {
	count uint64
	bytes uint64
	times [TIME_BUCKETS]uint64
}

var start = UnixNow()
var querycount int

var verbose = true
var noclean = false
var dirty = false

var stats struct {
	packets struct {
		rcvd     uint64
		rcvdsync uint64
	}
	desyncs uint64
	streams uint64
}

//UnixNow UnixNow
func UnixNow() int64 {
	return time.Now().Unix()
}

func calculateTimes(timings *[TIME_BUCKETS]uint64) (fmin, favg, fmax float64) {
	var counts, total, min, max, avg uint64 = 0, 0, 0, 0, 0
	hasmin := false
	for _, val := range *timings {
		if val == 0 {
			// Queries should never take 0 nanoseconds. We are using 0 as a
			// trigger to mean 'uninitialized reading'.
			continue
		}
		if val < min || !hasmin {
			hasmin = true
			min = val
		}
		if val > max {
			max = val
		}
		counts++
		total += val
	}
	if counts > 0 {
		avg = total / counts // integer division
	}
	return float64(min) / 1000000, float64(avg) / 1000000,
		float64(max) / 1000000
}

// Do something with a packet for a source.
func (h *MysqlDecode) processPacket(rs *source, request bool, data []byte) {
	// log.Infof("[%s] request=%t, got %d bytes", rs.src, request,
	// 	len(data))

	stats.packets.rcvd++
	if rs.synced {
		stats.packets.rcvdsync++
	}

	var ptype = -1
	var pdata []byte

	if request {
		// If we still have response buffer, we're in some weird state and
		// didn't successfully process the response.
		if rs.resbuffer != nil {
			//				log.Printf("[%s] possibly pipelined request? %d bytes",
			//					rs.src, len(rs.resbuffer))
			stats.desyncs++
			rs.resbuffer = nil
			rs.synced = false
		}
		rs.reqbuffer = data
		ptype, pdata = h.carvePacket(&rs.reqbuffer)
	} else {
		// FIXME: For now we're not doing anything with response data, just using the first packet
		// after a query to determine latency.
		rs.resbuffer = nil
		ptype, pdata = 0, data
	}

	// The synchronization logic: if we're not presently, then we want to
	// keep going until we are capable of carving off of a request/query.
	if !rs.synced {
		if !(request && ptype == COM_QUERY) {
			rs.reqbuffer, rs.resbuffer = nil, nil
			return
		}
		rs.synced = true
	}
	//log.Infof("[%s] request=%b ptype=%d plen=%d", rs.src, request, ptype, len(pdata))

	// No (full) packet detected yet. Continue on our way.
	if ptype == -1 {
		return
	}
	plen := uint64(len(pdata))

	// If this is a response then we want to record the timing and
	// store it with this channel so we can keep track of that.
	var reqtime uint64
	if !request {
		// Keep adding the bytes we're getting, since this is probably still part of
		// an earlier response
		if rs.reqSent == nil {
			if rs.qdata != nil {
				rs.qdata.bytes += plen
			}
			return
		}
		reqtime = uint64(time.Since(*rs.reqSent).Nanoseconds())
		if rs.qdata != nil {
			// This should never fail but it has. Probably because of a
			// race condition I need to suss out, or sharing between
			// two different goroutines. :(
			rs.qdata.bytes += plen
		}
		rs.reqSent = nil

		// If we're in verbose mode, just dump statistics from this one.
		if verbose && len(rs.qtext) > 0 {
			fmt.Printf("    %s%s %s## %sbytes: %d time: %0.2f%s\n", COLOR_GREEN, rs.qtext, COLOR_RED,
				COLOR_YELLOW, rs.qbytes, float64(reqtime)/1000000, COLOR_DEFAULT)
		}
		var code = "Success"
		if len(pdata) > 7 {
			if pdata[4] == 255 { //0xFF Error包

				code = "Error"
			}
			if pdata[4] == 254 { //0xFE EOF包
				code = "EOF"
			}
		}
		fmt.Println("Code:" + code)
		sqlinfo := strings.Split(rs.qtext, ":")
		var mm = metric.MysqlMessage{
			Code:          code,
			SQL:           sqlinfo[1],
			RemoteAddr:    sqlinfo[0],
			Reqtime:       reqtime,
			ContentLength: rs.qdata.bytes,
		}
		h.mysqlMetricStore.Input(mm)
		return
	}

	// This is for sure a request, so let's count it as one.
	if rs.reqSent != nil {
		//			log.Printf("[%s] ...sending two requests without a response?",
		//				rs.src)
	}
	tnow := time.Now()
	rs.reqSent = &tnow

	// Convert this request into whatever format the user wants.
	querycount++
	var text string

	for _, item := range h.format {
		switch item.(type) {
		case int:
			switch item.(int) {
			case F_NONE:
				log.Fatalf("F_NONE in format string")
			case F_QUERY:
				if dirty {
					text += string(pdata)
				} else {
					text += h.cleanupQuery(pdata)
				}
			case F_ROUTE:
				// Routes are in the query like:
				//     SELECT /* hostname:route */ FROM ...
				// We remove the hostname so routes can be condensed.
				parts := strings.SplitN(string(pdata), " ", 5)
				if len(parts) >= 4 && parts[1] == "/*" && parts[3] == "*/" {
					if strings.Contains(parts[2], ":") {
						text += strings.SplitN(parts[2], ":", 2)[1]
					} else {
						text += parts[2]
					}
				} else {
					text += "(unknown) " + h.cleanupQuery(pdata)
				}
			case F_SOURCE:
				text += rs.src
			case F_SOURCEIP:
				text += rs.srcip
			default:
				log.Fatalf("Unknown F_XXXXXX int in format string")
			}
		case string:
			text += item.(string)
		default:
			log.Fatalf("Unknown type in format string")
		}
	}

	qdata, ok := h.qbuf[text]
	if !ok {
		qdata = &queryData{}
		h.qbuf[text] = qdata
	}
	qdata.count++
	qdata.bytes += plen
	rs.qtext, rs.qdata, rs.qbytes = text, qdata, plen
}

// carvePacket tries to pull a packet out of a slice of bytes. If so, it removes
// those bytes from the slice.
func (h *MysqlDecode) carvePacket(buf *[]byte) (int, []byte) {
	datalen := uint32(len(*buf))
	if datalen < 5 {
		return -1, nil
	}

	size := uint32((*buf)[0]) + uint32((*buf)[1])<<8 + uint32((*buf)[2])<<16
	if size == 0 || datalen < size+4 {
		return -1, nil
	}

	// Else, has some length, try to validate it.
	end := size + 4
	ptype := int((*buf)[4])
	data := (*buf)[5 : size+4]
	if end >= datalen {
		*buf = nil
	} else {
		*buf = (*buf)[end:]
	}

	//	log.Printf("datalen=%d size=%d end=%d ptype=%d data=%d buf=%d",
	//		datalen, size, end, ptype, len(data), len(*buf))

	return ptype, data
}

// scans forward in the query given the current type and returns when we encounter
// a new type and need to stop scanning.  returns the size of the last token and
// the type of it.
func (h *MysqlDecode) scanToken(query []byte) (length int, thistype int) {
	if len(query) < 1 {
		log.Fatalf("scanToken called with empty query")
	}

	//no clean queries
	if verbose && noclean {
		return len(query), TOKEN_OTHER
	}
	// peek at the first byte, then loop
	b := query[0]
	switch {
	case b == 39 || b == 34: // '"
		startedwith := b
		escaped := false
		for i := 1; i < len(query); i++ {
			switch query[i] {
			case startedwith:
				if escaped {
					escaped = false
					continue
				}
				return i + 1, TOKEN_QUOTE
			case 92:
				escaped = true
			default:
				escaped = false
			}
		}
		return len(query), TOKEN_QUOTE

	case b >= 48 && b <= 57: // 0-9
		for i := 1; i < len(query); i++ {
			switch {
			case query[i] >= 48 && query[i] <= 57: // 0-9
				// do nothing
			default:
				return i, TOKEN_NUMBER
			}
		}
		return len(query), TOKEN_NUMBER

	case b == 32 || (b >= 9 && b <= 13): // whitespace
		for i := 1; i < len(query); i++ {
			switch {
			case query[i] == 32 || (query[i] >= 9 && query[i] <= 13):
				// Eat all whitespace
			default:
				return i, TOKEN_WHITESPACE
			}
		}
		return len(query), TOKEN_WHITESPACE

	case (b >= 65 && b <= 90) || (b >= 97 && b <= 122): // a-zA-Z
		for i := 1; i < len(query); i++ {
			switch {
			case query[i] >= 48 && query[i] <= 57:
				// Numbers, allow.
			case (query[i] >= 65 && query[i] <= 90) || (query[i] >= 97 && query[i] <= 122):
				// Letters, allow.
			case query[i] == 36 || query[i] == 95:
				// $ and _
			default:
				return i, TOKEN_WORD
			}
		}
		return len(query), TOKEN_WORD

	default: // everything else
		return 1, TOKEN_OTHER
	}
}

func (h *MysqlDecode) cleanupQuery(query []byte) string {
	// iterate until we hit the end of the query...
	var qspace []string
	for i := 0; i < len(query); {
		length, toktype := h.scanToken(query[i:])

		switch toktype {
		case TOKEN_WORD, TOKEN_OTHER:
			qspace = append(qspace, string(query[i:i+length]))

		case TOKEN_NUMBER, TOKEN_QUOTE:
			qspace = append(qspace, "?")

		case TOKEN_WHITESPACE:
			qspace = append(qspace, " ")

		default:
			log.Fatalf("scanToken returned invalid token type %d", toktype)
		}

		i += length
	}

	// Remove hostname from the route information if it's present
	tmp := strings.Join(qspace, "")

	parts := strings.SplitN(tmp, " ", 5)
	if len(parts) >= 5 && parts[1] == "/*" && parts[3] == "*/" {
		if strings.Contains(parts[2], ":") {
			tmp = parts[0] + " /* " + strings.SplitN(parts[2], ":", 2)[1] + " */ " + parts[4]
		}
	}

	return strings.Replace(tmp, "?, ", "", -1)
}

// parseFormat takes a string and parses it out into the given format slice
// that we later use to build up a string. This might actually be an overcomplicated
// solution?
func (h *MysqlDecode) parseFormat(formatstr string) {
	formatstr = strings.TrimSpace(formatstr)
	if formatstr == "" {
		formatstr = "#b:#k"
	}

	isspecial := false
	curstr := ""
	doappend := F_NONE
	for _, char := range formatstr {
		if char == '#' {
			if isspecial {
				curstr += string(char)
				isspecial = false
			} else {
				isspecial = true
			}
			continue
		}

		if isspecial {
			switch strings.ToLower(string(char)) {
			case "s":
				doappend = F_SOURCE
			case "i":
				doappend = F_SOURCEIP
			case "r":
				doappend = F_ROUTE
			case "q":
				doappend = F_QUERY
			default:
				curstr += "#" + string(char)
			}
			isspecial = false
		} else {
			curstr += string(char)
		}

		if doappend != F_NONE {
			if curstr != "" {
				h.format = append(h.format, curstr, doappend)
				curstr = ""
			} else {
				h.format = append(h.format, doappend)
			}
			doappend = F_NONE
		}
	}
	if curstr != "" {
		h.format = append(h.format, curstr)
	}
}

//Len Len
func (s sortableSlice) Len() int {
	return len(s)
}

//Less Less
func (s sortableSlice) Less(i, j int) bool {
	return s[i].value < s[j].value
}

//Swap Swap
func (s sortableSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
