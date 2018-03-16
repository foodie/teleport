package main

import (
	"fmt"
	"time"
)

func main() {
	c1 := make(chan int)
	c2 := make(chan int)
	go func() {
		c1 <- 1

		time.Sleep(time.Second * 30)
	}()
	go func() {
		c2 <- 1
		time.Sleep(time.Second * 20)
	}()
	select {
	case <-c1:
		fmt.Println("c1")

	case <-c2:
		fmt.Println("c1")
	}
	time.Sleep(time.Second * 2)
	fmt.Println("end")
}
