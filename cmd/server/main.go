package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config, err := ParseConfig(os.Args[1:], os.Getenv)
	if err != nil {
		log.Printf("配置错误: %v", err)
		os.Exit(2)
	}
	if config.SelfCheck {
		if err := RunSelfCheck(config); err != nil {
			log.Printf("自检失败: %v", err)
			os.Exit(1)
		}
		fmt.Println("selfcheck ok: 主流程、证据摘要与凭据验真通过")
		return
	}
	runtime, err := BuildRuntime(config)
	if err != nil {
		log.Printf("启动失败: %v", err)
		os.Exit(1)
	}
	errors := make(chan error, 1)
	go runtime.Serve(errors)
	log.Printf("seed-vigor-gate 已监听 http://%s", config.Address)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errors:
		if err != nil {
			log.Printf("HTTP 服务异常: %v", err)
		}
		_ = runtime.Store.Close()
		if err != nil {
			os.Exit(1)
		}
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := runtime.Shutdown(ctx); err != nil {
			log.Printf("优雅关闭失败: %v", err)
			os.Exit(1)
		}
	}
}
