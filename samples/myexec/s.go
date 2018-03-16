package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {

	//监听端口
	svr := tp.NewPeer(tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	})
	//注册route
	svr.RoutePull(new(math))
	svr.Listen()
}

type math struct {
	tp.PullCtx
}

//处理数据
func (m *math) Add(args *[]int) (int, *tp.Rerror) {
	if m.Query().Get("push_status") == "yes" {
		m.Session().Push(
			"/push/status",
			fmt.Sprintf("numbers %+v are being added...", args),
		)
		time.Sleep(time.Millisecond * 10)
	}
	var r int
	for _, a := range *args {
		r += a
	}
	return r, nil
}
