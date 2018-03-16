package utils

import (
	"io"
	"sort"
	"sync"
	"sync/atomic"
)

// ByteBuffer provides byte buffer, which can be used for minimizing
// memory allocations.
//
// ByteBuffer may be used with functions appending data to the given []byte
// slice. See example code for details.
//
// Use Get for obtaining an empty byte buffer.
//有bufer的byte
type ByteBuffer struct {

	// B is a byte buffer to use in append-like workloads.
	// See example code for details.
	B []byte
}

//长度
// Len returns the size of the byte buffer.
func (b *ByteBuffer) Len() int {
	return len(b.B)
}

// ReadFrom implements io.ReaderFrom.
//
//从r读取数据到buf
// The function appends all the data read from r to b.
func (b *ByteBuffer) ReadFrom(r io.Reader) (int64, error) {
	p := b.B

	nStart := int64(len(p)) //当前的长度
	nMax := int64(cap(p))   //容量
	n := nStart
	if nMax == 0 {
		nMax = 64 //默认为64
		p = make([]byte, nMax)
	} else {
		p = p[:nMax] //p不能超过max
	}
	//p是未用的空间
	for {
		if n == nMax { //如果到达容量极限，按照两倍扩容
			nMax *= 2
			bNew := make([]byte, nMax)
			copy(bNew, p)
			p = bNew
		}
		//读取数据
		nn, err := r.Read(p[n:]) //把数据放入p
		n += int64(nn)
		if err != nil { //读取错误
			b.B = p[:n]        //p拷贝到b.B
			n -= nStart        //已经读取的长度
			if err == io.EOF { //错误
				return n, nil
			}
			return n, err
		}
	}
}

// Bytes returns b.B, i.e. all the bytes accumulated in the buffer.
//返回byte
// The purpose of this function is bytes.Buffer compatibility.
func (b *ByteBuffer) Bytes() []byte {
	return b.B
}

//把byte写入ByteBuffer
// Write implements io.Writer - it appends p to ByteBuffer.B
func (b *ByteBuffer) Write(p []byte) (int, error) {
	b.B = append(b.B, p...)
	return len(p), nil
}

// WriteByte appends the byte c to the buffer.
//
// The purpose of this function is bytes.Buffer compatibility.
//写一个byte到ByteBuffer
// The function always returns nil.
func (b *ByteBuffer) WriteByte(c byte) error {
	b.B = append(b.B, c)
	return nil
}

//写string
// WriteString appends s to ByteBuffer.B.
func (b *ByteBuffer) WriteString(s string) (int, error) {
	b.B = append(b.B, s...)
	return len(s), nil
}

//重写设置 to byte[]
// Set sets ByteBuffer.B to p.
func (b *ByteBuffer) Set(p []byte) {
	b.B = append(b.B[:0], p...)
}

//重写设置 to string
// SetString sets ByteBuffer.B to s.
func (b *ByteBuffer) SetString(s string) {
	b.B = append(b.B[:0], s...)
}

//转换成string
// String returns string representation of ByteBuffer.B.
func (b *ByteBuffer) String() string {
	return string(b.B)
}

//重置
// Reset makes ByteBuffer.B empty.
func (b *ByteBuffer) Reset() {
	b.B = b.B[:0]
}

//重置长度
// ChangeLen changes the buffer length.
func (b *ByteBuffer) ChangeLen(newLen int) {
	if cap(b.B) < newLen {
		b.B = make([]byte, newLen)
	} else {
		b.B = b.B[:newLen]
	}
}

const (

	//最小bit
	minBitSize = 6 // 2**6=64 is a CPU cache line size

	//step
	steps = 20

	//1<<6 最小size，2**6 = 64
	minSize = 1 << minBitSize
	//2**25 ---- 32M
	maxSize = 1 << (minBitSize + steps - 1)

	//每一个hash存储的数据的最大量
	calibrateCallsThreshold = 42000

	//因子
	maxPercentile = 0.95
)

// BufferPool represents byte buffer pool.
//
// Distinct pools may be used for distinct types of byte buffers.
// Properly determined byte buffer types with their own pools may help reducing
// memory waste.
//buffer池
type BufferPool struct {
	calls       [steps]uint64 //calls [20]uint64
	calibrating uint64        //是否检验过

	defaultSize uint64 //默认尺寸
	maxSize     uint64 //最大尺寸

	pool sync.Pool //系统的pool
}

//默认的pool
var defaultBufferPool BufferPool

// AcquireByteBuffer returns an empty byte buffer from the pool.
//
// Got byte buffer may be returned to the pool via Put call.
// This reduces the number of memory allocations required for byte buffer
// management.
//获取一个ByteBuffer
func AcquireByteBuffer() *ByteBuffer {
	return defaultBufferPool.Get()
}

// Get returns new byte buffer with zero length.
//
// The byte buffer may be returned to the pool via Put after the use
// in order to minimize GC overhead.
//创建一个新的buffer
//有就返回ByteBuffer，没有就返回一个默认大小的[]byte的defaultSize
func (p *BufferPool) Get() *ByteBuffer {
	v := p.pool.Get()
	if v != nil {
		return v.(*ByteBuffer)
	}
	return &ByteBuffer{
		B: make([]byte, 0, atomic.LoadUint64(&p.defaultSize)),
	}
}

// ReleaseByteBuffer returns byte buffer to the pool.
//
// ByteBuffer.B mustn't be touched after returning it to the pool.
// Otherwise data races will occur.
//放置ByteBuffer
func ReleaseByteBuffer(b *ByteBuffer) {
	defaultBufferPool.Put(b)
}

// Put releases byte buffer obtained via Get to the pool.
//
// The buffer mustn't be accessed after returning to the pool.
func (p *BufferPool) Put(b *ByteBuffer) {

	//默认的buffer的 idx大小
	idx := index(len(b.B))

	//calls[idx] ==calibrateCallsThreshold
	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	/**
	 n < calibrateCallsThreshold 前maxSize都是0
	 直接重置，放入即可
	**/
	maxSize := int(atomic.LoadUint64(&p.maxSize))
	//如果maxSize为0 或者cap(b.B)小于等于maxSize

	//容量小于maxSize
	//把b重置后，放入pool里面
	if maxSize == 0 || cap(b.B) <= maxSize {
		b.Reset()     //重置数据
		p.pool.Put(b) //放入数据
	}
}

//校准
func (p *BufferPool) calibrate() {
	//是否已经校准过
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	//新建一个callSizes的数组
	a := make(callSizes, 0, steps)

	//调用的总次数
	var callsSum uint64

	//获取a的数据
	for i := uint64(0); i < steps; i++ {
		//将0保存到p.calls[i]并返回旧值
		calls := atomic.SwapUint64(&p.calls[i], 0)
		callsSum += calls
		a = append(a, callSize{
			calls: calls,        //调用次数
			size:  minSize << i, // 最大值
		})
	}
	//排序
	sort.Sort(a)

	//最小尺寸
	defaultSize := a[0].size

	//默认的最大尺寸
	maxSize := defaultSize

	//调用总数量
	maxSum := uint64(float64(callsSum) * maxPercentile)

	//调用总数
	callsSum = 0
	for i := 0; i < steps; i++ {
		if callsSum > maxSum {
			break
		}
		// 不断加
		callsSum += a[i].calls
		size := a[i].size
		if size > maxSize {
			maxSize = size
		}
	}

	//设置defaultSize， maxSize
	atomic.StoreUint64(&p.defaultSize, defaultSize)
	atomic.StoreUint64(&p.maxSize, maxSize)
	//设置校验完成
	atomic.StoreUint64(&p.calibrating, 0)
}

//callsize是可以排序的
type callSize struct {
	calls uint64
	size  uint64
}

type callSizes []callSize

func (ci callSizes) Len() int {
	return len(ci)
}

func (ci callSizes) Less(i, j int) bool {
	return ci[i].calls > ci[j].calls
}

func (ci callSizes) Swap(i, j int) {
	ci[i], ci[j] = ci[j], ci[i]
}

//index 是个简单的hash 从0到step-1
//确定落点
func index(n int) int {
	n--
	n >>= minBitSize
	idx := 0
	for n > 0 {
		n >>= 1
		idx++
	}
	if idx >= steps {
		idx = steps - 1
	}
	return idx
}
