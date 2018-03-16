// Copyright 2017 HenryLee. All Rights Reserved.
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

package xfer

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/henrylee2cn/teleport/utils"
)

func init() {
	Reg(newGzip('g', 5))
}

//新建一个压缩器
func newGzip(id byte, level int) *Gzip {
	if level < gzip.HuffmanOnly || level > gzip.BestCompression {
		panic(fmt.Sprintf("gzip: invalid compression level: %d", level))
	}
	g := new(Gzip)
	g.level = level
	g.id = id

	//指定了压缩水平而不是采用默认的DefaultCompression。
	//创建并返回一个Writer
	g.wPool = sync.Pool{
		New: func() interface{} {
			gw, _ := gzip.NewWriterLevel(nil, g.level)
			return gw
		},
	}

	//Reader类型满足io.Reader接口，可以从gzip格式压缩文件读取并解压数据。
	g.rPool = sync.Pool{
		New: func() interface{} {
			return new(gzip.Reader)
		},
	}
	return g
}

//定义gzip压缩过滤器
// Gzip compression filter
type Gzip struct {
	id    byte
	level int
	wPool sync.Pool
	rPool sync.Pool
}

//获取id
// Id returns transfer filter id.
func (g *Gzip) Id() byte {
	return g.id
}

//压缩数据
// OnPack performs filtering on packing.
func (g *Gzip) OnPack(src []byte) ([]byte, error) {
	//获取写gw
	gw := g.wPool.Get().(*gzip.Writer)
	//放入gw
	defer g.wPool.Put(gw)

	//获取一个空的buffer
	bb := utils.AcquireByteBuffer()
	gw.Reset(bb)
	//压缩写入src
	_, err := gw.Write(src)
	if err != nil {
		utils.ReleaseByteBuffer(bb)
		return nil, err
	}
	//flush
	err = gw.Flush()
	if err != nil {
		utils.ReleaseByteBuffer(bb)
		return nil, err
	}
	//从bb获取数据
	return bb.Bytes(), nil
}

//解压数据
// OnUnpack performs filtering on unpacking.
func (g *Gzip) OnUnpack(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	gr := g.rPool.Get().(*gzip.Reader)
	defer g.rPool.Put(gr)
	err := gr.Reset(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	//压缩读取
	dest, _ := ioutil.ReadAll(gr)
	return dest, nil
}
