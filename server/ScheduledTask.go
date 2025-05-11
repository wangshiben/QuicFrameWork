package server

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

//3分钟检测一次session大小
//const memoTickerTime = time.Second * 3

func (s *Server) ScheduledTask() {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	go func() {
		ticker := time.NewTicker(s.Session.GetNextTimePicker())
		//ticker := time.NewTicker(time.Second * 2)

		defer ticker.Stop()

		// 创建一个接收操作系统信号的通道，用于处理中断（Ctrl+C）
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		flag := true

		// 启动一个goroutine来处理ticker发送的时间

		for flag {
			select {
			case _ = <-sigChan:
				flag = false
				break
			case _ = <-ticker.C:
				//sizeof := size.Of(s.Session)
				s.Session.CleanExpItem()
				//fmt.Println("目前session大小: " + fmt.Sprintf("%d", sizeof))
			}
		}

	}()

}

// 关闭监听退出服务器
func (s *Server) closeListener() {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		s.Server.Close()
		s.quicServer.Close()
	}()
}
