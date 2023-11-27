package tcp

/**
 * A echo server to test whether the server is functioning normally
 */

import (
	"bufio"
	"context"
	"github.com/hdt3213/godis/lib/logger"
	"github.com/hdt3213/godis/lib/sync/atomic"
	"github.com/hdt3213/godis/lib/sync/wait"
	"io"
	"net"
	"sync"
	"time"
)

// EchoHandler 会将接收到的行返回给客户端，用于测试
type EchoHandler struct {
	// 保存所有工作状态 client 的集合(把 map 当 set 用)
	// 使用并发安全的容器，然后往里面塞 tcp 连接（这样就能安全的存储了）
	activeConn sync.Map

	// 关闭状态标识位
	closing atomic.Boolean
}

// MakeEchoHandler 用于创建 EchoHandler
func MakeEchoHandler() *EchoHandler {
	return &EchoHandler{}
}

// EchoClient 是 EchoHandler 的客户端，用于测试
type EchoClient struct {
	Conn    net.Conn
	Waiting wait.Wait
}

// Close 关闭客户端连接
func (c *EchoClient) Close() error {
	c.Waiting.WaitWithTimeout(10 * time.Second)
	c.Conn.Close()
	return nil
}

// Handle 给客户端返回响应行
func (h *EchoHandler) Handle(ctx context.Context, conn net.Conn) {
	// 关闭中的 handler 不会处理新连接
	if h.closing.Get() {
		_ = conn.Close()
		return
	}

	client := &EchoClient{
		Conn: conn,
	}
	h.activeConn.Store(client, struct{}{}) // 记住仍然存活的连接

	reader := bufio.NewReader(conn)
	for {
		// 从客户端读取数据
		msg, err := reader.ReadString('\n')
		// 可能发生的错误：客户端关闭（err == EOF）、客户端超时、服务器提前关闭
		if err != nil {
			if err == io.EOF {
				logger.Info("connection close")
				h.activeConn.Delete(client)
			} else {
				logger.Warn(err)
			}
			return
		}
		// 发送数据前先置为 waiting 状态，阻止连接被关闭
		client.Waiting.Add(1)

		// 模拟关闭时未完成发送的情况
		// logger.Info("sleeping")
		// time.Sleep(10 * time.Second)

		// 写入数据
		b := []byte(msg)
		_, _ = conn.Write(b)

		// 发送完毕, 结束 waiting
		client.Waiting.Done()
	}
}

// Close 关闭服务器
func (h *EchoHandler) Close() error {
	logger.Info("handler shutting down...")
	h.closing.Set(true)
	// 逐个关闭连接
	h.activeConn.Range(func(key interface{}, val interface{}) bool {
		client := key.(*EchoClient)
		_ = client.Close()
		return true
	})
	return nil
}
