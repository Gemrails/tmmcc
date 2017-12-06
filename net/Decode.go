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
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

//SourceData 解码前数据
type SourceData struct {
	Source      []byte
	ReceiveDate time.Time
	SourcePoint *gopacket.Endpoint
	TargetPoint *gopacket.Endpoint
	SourceHost  *gopacket.Endpoint
	TargetHost  *gopacket.Endpoint
	TCP         *layers.TCP
}

//Decode 不同协议解码器接口
type Decode interface {
	Decode(*SourceData)
}

//FindDecode 通过协议查找解码器
func FindDecode(protol string) Decode {
	switch protol {
	case "http":
		return &HTTPDecode{}
	case "mysql":
		return &MysqlDecode{}
	default:
		return nil
	}
}
