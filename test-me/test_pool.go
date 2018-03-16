package main

import (
	"log"
	"sort"
	"sync"

	"weibo.com/READ_CODE/tcp/other/teleport/test/utils"
)

const (
	minBitSize = 6
	steps      = 20
)

func main() {
	//test_pool()
	testByteBuf()
	test12()
}

func test12() {

	arr := []int{11, 32, 3, 14, 5}
	log.Println(arr)
	sort.Ints(arr)
	log.Println(arr)
}

func testByteBuf() {
	b := new(utils.ByteBuffer)
	b.WriteString("1afawoerieworewo")
	log.Println(b.Len())

	b2 := new(utils.ByteBuffer)
	b2.WriteString("1afawoerieworewo")
	log.Println(b2.Len())

	utils.ReleaseByteBuffer(b)
	utils.ReleaseByteBuffer(b2)

	b3 := utils.AcquireByteBuffer()
	log.Println(b3.Len())

	b4 := utils.AcquireByteBuffer()
	log.Println(b4.Len())

}

func test_pool() {

	// 建立对象
	var pipe = &sync.Pool{
		New: func() interface{} { return "Hello,BeiJing" }}
	// 准备放入的字符串
	val := "Hello,World!"
	// 放入
	pipe.Put(val)
	// 取出
	log.Println(pipe.Get())
	// 再取就没有了,会自动调用NEW
	log.Println(pipe.Get())

	//测试slice
	b := make([]byte, 5)
	b = append(b, []byte("ab")...)
	log.Println(len(b))
	b = b[:0]
	log.Println(len(b))
}
