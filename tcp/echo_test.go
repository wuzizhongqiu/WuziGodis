package tcp

import (
	"bufio"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"
)

// TestListenAndServe 单元测试代码
func TestListenAndServe(t *testing.T) {
	var err error
	closeChan := make(chan struct{})
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Error(err)
		return
	}
	addr := listener.Addr().String()
	go ListenAndServe(listener, MakeEchoHandler(), closeChan) // 启动 TCP 服务器

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Error(err)
		return
	}
	// 随机写入一些值
	for i := 0; i < 10; i++ {
		val := strconv.Itoa(rand.Int())
		_, err = conn.Write([]byte(val + "\n"))
		if err != nil {
			t.Error(err)
			return
		}
		bufReader := bufio.NewReader(conn)
		line, _, err := bufReader.ReadLine()
		if err != nil {
			t.Error(err)
			return
		}
		if string(line) != val {
			t.Error("get wrong response")
			return
		}
	}
	_ = conn.Close()
	for i := 0; i < 5; i++ {
		// 创建一些空闲的连接
		_, _ = net.Dial("tcp", addr)
	}
	closeChan <- struct{}{}
	time.Sleep(time.Second)
}
