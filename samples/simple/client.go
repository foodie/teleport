package main

import (
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	//日志级别
	tp.SetLoggerLevel("ERROR")
	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	//定义路由
	cli.RoutePush(new(push))
	//发起请求
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

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	//返回结果
	tp.Printf("reply: %d", reply)
}

type push struct {
	tp.PushCtx
}

//获取状态
func (p *push) Status(args *string) *tp.Rerror {
	tp.Printf("server status: %s", *args)
	return nil
}
