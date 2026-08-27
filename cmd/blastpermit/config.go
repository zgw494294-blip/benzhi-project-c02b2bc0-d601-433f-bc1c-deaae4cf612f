package main

import (
	"flag"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
)

type config struct {
	addr      string
	database  string
	selfcheck bool
}

func parseConfig() (config, error) {
	defaultAddr := "127.0.0.1:19081"
	if raw := os.Getenv("PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return config{}, fmt.Errorf("PORT 必须是 1 到 65535 的端口号")
		}
		defaultAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	var c config
	flag.StringVar(&c.addr, "addr", defaultAddr, "HTTP 监听地址（必须为回环地址）")
	flag.StringVar(&c.database, "db", "blast-permit.db", "SQLite 数据库路径")
	flag.BoolVar(&c.selfcheck, "selfcheck", false, "运行完整 HTTP 自检后退出")
	flag.Parse()
	if err := validateAddr(c.addr); err != nil {
		return config{}, err
	}
	return c, nil
}
func validateAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("addr 必须为 host:port: %w", err)
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !ip.IsLoopback() {
		return fmt.Errorf("addr 必须使用明确的回环 IP 地址")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("addr 端口无效")
	}
	return nil
}
