package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const defaultAddress = "127.0.0.1:19081"

type Config struct {
	Address      string
	DatabasePath string
	SelfCheck    bool
}

func ParseConfig(args []string, getenv func(string) string) (Config, error) {
	set := flag.NewFlagSet("seed-vigor-gate", flag.ContinueOnError)
	set.SetOutput(discardWriter{})
	var address, database string
	var selfcheck bool
	set.StringVar(&address, "addr", "", "HTTP 监听地址")
	set.StringVar(&database, "db", "./data/seed-vigor-gate.db", "SQLite 数据文件")
	set.BoolVar(&selfcheck, "selfcheck", false, "运行主流程自检并退出")
	if err := set.Parse(args); err != nil {
		return Config{}, err
	}
	if len(set.Args()) != 0 {
		return Config{}, fmt.Errorf("不支持的位置参数: %s", strings.Join(set.Args(), " "))
	}
	if address == "" {
		portValue := strings.TrimSpace(getenv("PORT"))
		if portValue == "" {
			address = defaultAddress
		} else {
			port, err := strconv.Atoi(portValue)
			if err != nil || port < 1 || port > 65535 {
				return Config{}, fmt.Errorf("PORT 必须是 1 至 65535 的端口号")
			}
			address = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		}
	}
	if err := validateAddress(address); err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(database) == "" {
		return Config{}, fmt.Errorf("-db 不能为空")
	}
	return Config{Address: address, DatabasePath: database, SelfCheck: selfcheck}, nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("-addr 必须为 host:port: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("-addr 端口无效")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("-addr 只允许显式回环 IP，拒绝 %q", host)
	}
	return nil
}

type discardWriter struct{}

func (discardWriter) Write(value []byte) (int, error) { return len(value), nil }
