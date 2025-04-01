package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// 获取本地IP地址，返回所有192.168开头的IP
func getLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				if strings.HasPrefix(ip, "192.168") {
					ips = append(ips, ip)
				}
			}
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("没有找到192.168开头的IP地址")
	}
	return ips, nil
}

// 带重试的IP获取函数
func getLocalIPsWithRetry(timeout time.Duration) ([]string, error) {
	startTime := time.Now()
	for {
		ips, err := getLocalIPs()
		if err == nil {
			return ips, nil
		}

		// 检查是否超时
		if time.Since(startTime) >= timeout {
			return nil, fmt.Errorf("在%v时间内未能获取到IP地址", timeout)
		}

		// 等待5秒后重试
		time.Sleep(5 * time.Second)
		fmt.Println("未找到IP，5秒后重试...")
	}
}

// 发送Gotify通知
func sendGotifyNotification(serverURL, token, title, message string) error {
	values := url.Values{}
	values.Set("title", title)
	values.Set("message", message)
	values.Set("priority", "5")

	reqURL := fmt.Sprintf("%s/message?token=%s", serverURL, token)
	resp, err := http.PostForm(reqURL, values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("发送通知失败，状态码：%d", resp.StatusCode)
	}
	return nil
}

func main() {
	// 获取hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "未知主机"
		fmt.Printf("获取主机名失败: %v\n", err)
	}

	// 读取配置文件
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("无法加载配置文件: %v\n", err)
		return
	}

	// 获取Gotify配置
	gotifyServer := cfg.Section("gotify").Key("server_url").String()
	gotifyToken := cfg.Section("gotify").Key("token").String()

	// 获取所有本地IP，最多尝试1分钟
	ips, err := getLocalIPsWithRetry(1 * time.Minute)

	var message string
	var title string

	if err != nil {
		title = fmt.Sprintf("[%s] IP地址获取失败", hostname)
		message = fmt.Sprintf("获取IP地址失败: %v", err)
	} else {
		title = fmt.Sprintf("[%s] IP地址通知", hostname)
		message = fmt.Sprintf("主机：%s\n发现以下IP地址：\n", hostname)
		for _, ip := range ips {
			message += fmt.Sprintf("- %s\n", ip)
		}
	}

	// 发送通知
	err = sendGotifyNotification(
		gotifyServer,
		gotifyToken,
		title,
		message,
	)
	if err != nil {
		fmt.Printf("发送通知失败: %v\n", err)
		return
	}

	fmt.Println("通知发送成功！")
}
