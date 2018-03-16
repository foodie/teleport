package main

import (
	"compress/gzip"
	"fmt"
)

func main() {

	size := 1
	datasSlice := make([][]byte, size)
	for i := 0; i < size; i++ {
		datasSlice[i], _ = ioutil.ReadFile("images/" + strconv.Itoa(i+1) + ".jpg")
		fmt.Println("raw size:", strconv.Itoa(i)+".jpg :", len(datasSlice[i]))
	}

	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	defer w.Close()

	for i := 0; i < size; i++ {
		w.Write(datasSlice[i])
	}
	w.Flush()
	fmt.Println("gzip size:", len(b.Bytes()))

	r, _ := gzip.NewReader(&b)
	defer r.Close()
	undatas, _ := ioutil.ReadAll(r)
	fmt.Println("ungzip size:", len(undatas))

}
