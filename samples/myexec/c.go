package main

import (
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("ERROR")

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	cli.RoutePush(new(ct))

	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}
	//发起请求
	var reply int
	rerr := sess.Pull("/math/add?push_status=yes",
		[]int{1, 2, 3, 4, 5},
		&reply,
	).Rerror()
	//
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	//返回结果
	tp.Printf("reply: %d", reply)

}

type ct struct {
	tp.PushCtx
}

//获取状态
func (p *ct) Status(args *string) *tp.Rerror {
	tp.Printf("server status: %s", *args)
	return nil
}
