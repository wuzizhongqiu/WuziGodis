package tcp

/**
 * A tcp server
 */

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hdt3213/godis/interface/tcp"
	"github.com/hdt3213/godis/lib/logger"
)

// Config 存储 TCP 服务器属性
type Config struct {
	Address    string        `yaml:"address"`
	MaxConnect uint32        `yaml:"max-connect"`
	Timeout    time.Duration `yaml:"timeout"`
}

// ClientCounter 记录当前 WuziGodis 服务器上的客户端数量
var ClientCounter int

// ListenAndServe 监听并提供服务，并在收到 closeChan 发来的关闭通知后关闭
func ListenAndServe(listener net.Listener, handler tcp.Handler, closeChan <-chan struct{}) {
	errCh := make(chan error, 1)
	defer close(errCh)
	// 监听关闭通知
	go func() {
		select {
		case <-closeChan:
			logger.Info("get exit signal")
		case er := <-errCh:
			logger.Info(fmt.Sprintf("accept error: %s", er.Error()))
		}
		logger.Info("shutting down...")
		_ = listener.Close() // 停止监听，listener.Accept()会立即返回 io.EOF
		_ = handler.Close()  // 关闭应用层服务器
	}()

	ctx := context.Background() // 没用上
	var waitDone sync.WaitGroup
	for {
		// 监听端口, 阻塞直到收到新连接或者出现错误
		conn, err := listener.Accept()
		if err != nil { // 错误处理
			// learn from net/http/serve.go#Serve()
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				logger.Info("accept occurs temporary error: %v, retry in 5ms", err)
				time.Sleep(5 * time.Millisecond)
				continue
			}
			errCh <- err
			break
		}

		// 开启 goroutine 来处理新连接
		logger.Info("accept link")
		ClientCounter++
		waitDone.Add(1)
		go func() {
			defer func() {
				waitDone.Done()
				ClientCounter--
			}()
			handler.Handle(ctx, conn)
		}()
	}
	waitDone.Wait()
}

// ListenAndServeWithSignal 监听中断信号并通过 closeChan 通知服务器关闭（main 包调用的就是这个函数）
func ListenAndServeWithSignal(cfg *Config, handler tcp.Handler) error {
	// 优雅退出的实现
	closeChan := make(chan struct{})
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		switch sig {
		case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			closeChan <- struct{}{}
		}
	}()
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}
	// cfg.Address = listener.Addr().String()
	logger.Info(fmt.Sprintf("bind: %s, start listening...", cfg.Address))
	ListenAndServe(listener, handler, closeChan)
	return nil
}
