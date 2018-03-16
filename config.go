// Copyright 2015-2017 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tp

import (
	"errors"
	"math"
	"time"

	"github.com/henrylee2cn/cfgo"
	"github.com/henrylee2cn/teleport/socket"
)

// PeerConfig peer config
// Note:
//  yaml tag is used for github.com/henrylee2cn/cfgo
//  ini tag is used for github.com/henrylee2cn/ini
type PeerConfig struct {
	//类型
	Network       string `yaml:"network"              ini:"network"              comment:"Network; tcp, tcp4, tcp6, unix or unixpacket"`
	ListenAddress string `yaml:"listen_address"       ini:"listen_address"       comment:"Listen address; for server role"`
	//dail超时
	DefaultDialTimeout time.Duration `yaml:"default_dial_timeout" ini:"default_dial_timeout" comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
	//重试次数
	RedialTimes int32 `yaml:"redial_times"         ini:"redial_times"         comment:"The maximum times of attempts to redial, after the connection has been unexpectedly broken; for client role"`
	//codec type
	DefaultBodyCodec string `yaml:"default_body_codec"   ini:"default_body_codec"   comment:"Default body codec type id"`
	//session存在时间
	DefaultSessionAge time.Duration `yaml:"default_session_age"  ini:"default_session_age"  comment:"Default session max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
	//PULL or PUSH context max age
	DefaultContextAge time.Duration `yaml:"default_context_age"  ini:"default_context_age"  comment:"Default PULL or PUSH context max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
	//慢操作报警
	SlowCometDuration time.Duration `yaml:"slow_comet_duration"  ini:"slow_comet_duration"  comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
	//打印
	PrintBody bool `yaml:"print_body"           ini:"print_body"           comment:"Is print body or not"`
	//计算时间
	CountTime bool `yaml:"count_time"           ini:"count_time"           comment:"Is count cost time or not"`

	slowCometDuration time.Duration
}

//默认配置
var _ cfgo.Config = new(PeerConfig)

//加载配置
// Reload Bi-directionally synchronizes config between YAML file and memory.
func (p *PeerConfig) Reload(bind cfgo.BindFunc) error {
	err := bind()
	if err != nil {
		return err
	}
	return p.check()
}

//检测是否合法
func (p *PeerConfig) check() error {
	switch p.Network {
	default:
		return errors.New("Invalid network config, refer to the following: tcp, tcp4, tcp6, unix or unixpacket.")
	case "":
		p.Network = "tcp"
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
	}
	p.slowCometDuration = math.MaxInt64
	if p.SlowCometDuration > 0 {
		p.slowCometDuration = p.SlowCometDuration
	}
	//默认的
	if len(p.DefaultBodyCodec) == 0 {
		p.DefaultBodyCodec = "json"
	}
	return nil
}

// DefaultProtoFunc gets the default builder of socket communication protocol
//  func DefaultProtoFunc() socket.ProtoFunc
//默认proto func
var DefaultProtoFunc = socket.DefaultProtoFunc

// SetDefaultProtoFunc sets the default builder of socket communication protocol
//  func SetDefaultProtoFunc(protoFunc socket.ProtoFunc)
//设置proto func
var SetDefaultProtoFunc = socket.SetDefaultProtoFunc

// GetReadLimit gets the packet size upper limit of reading.
//  GetReadLimit() uint32
//获取读的包大小限制
var GetReadLimit = socket.PacketSizeLimit

// SetReadLimit sets max packet size.
// If maxSize<=0, set it to max uint32.
//  func SetReadLimit(maxPacketSize uint32)
//设置读的包大小限制
var SetReadLimit = socket.SetPacketSizeLimit

// SetSocketKeepAlive sets whether the operating system should send
// keepalive messages on the connection.
// Note: If have not called the function, the system defaults are used.
//  func SetSocketKeepAlive(keepalive bool)
//设置包的keeplive
var SetSocketKeepAlive = socket.SetKeepAlive

// SetSocketKeepAlivePeriod sets period between keep alives.
// Note: if d<0, don't change the value.
//  func SetSocketKeepAlivePeriod(d time.Duration)
//保持长连接的频次
var SetSocketKeepAlivePeriod = socket.SetKeepAlivePeriod

// SocketReadBuffer returns the size of the operating system's
// receive buffer associated with the connection.
// Note: if using the system default value, bytes=-1 and isDefault=true.
//  func SocketReadBuffer() (bytes int, isDefault bool)
//读的socket buffer
var SocketReadBuffer = socket.ReadBuffer

// SetSocketReadBuffer sets the size of the operating system's
// receive buffer associated with the connection.
// Note: if bytes<0, don't change the value.
//  func SetSocketReadBuffer(bytes int)
//设置读的socket buffer
var SetSocketReadBuffer = socket.SetReadBuffer

// SocketWriteBuffer returns the size of the operating system's
// transmit buffer associated with the connection.
// Note: if using the system default value, bytes=-1 and isDefault=true.
//  func SocketWriteBuffer() (bytes int, isDefault bool)
//写的socket buffer
var SocketWriteBuffer = socket.WriteBuffer

// SetSocketWriteBuffer sets the size of the operating system's
// transmit buffer associated with the connection.
// Note: if bytes<0, don't change the value.
//  func SetSocketWriteBuffer(bytes int)
//设置写的socket buffer
var SetSocketWriteBuffer = socket.SetWriteBuffer

// SetSocketNoDelay controls whether the operating system should delay
// packet transmission in hopes of sending fewer packets (Nagle's
// algorithm).  The default is true (no delay), meaning that data is
// sent as soon as possible after a Write.
//  func SetSocketNoDelay(noDelay bool)
//不延时
var SetSocketNoDelay = socket.SetNoDelay
