package ports

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// FindPIDByPort 查找占用指定端口的PID
func FindPIDByPort(port string) (string, error) {
	cmd := exec.Command("netstat", "-ano")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("执行 netstat 失败: %v", err)
	}

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			localAddress := fields[1]
			if strings.Contains(localAddress, ":"+port) {
				return fields[4], nil // 返回PID
			}
		}
	}

	return "", fmt.Errorf("未找到占用端口 %s 的进程", port)
}

// KillProcessByPID 杀掉指定PID的进程
func KillProcessByPID(pid string) error {
	cmd := exec.Command("taskkill", "/PID", pid, "/F")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("无法杀掉PID %s: %v", pid, err)
	}
	//fmt.Printf("成功杀掉 PID %s 的进程\n", pid)

	// 清理 DNS 缓存，确保网络不受干扰
	if err := FlushDNS(); err != nil {
		//fmt.Printf("清理 DNS 缓存失败: %v\n", err)
	} else {
		//fmt.Println("成功清理 DNS 缓存")
	}
	return nil
}

// FlushDNS 刷新 DNS 缓存
func FlushDNS() error {
	cmd := exec.Command("ipconfig", "/flushdns")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("无法刷新 DNS 缓存: %v", err)
	}
	return nil
}

// FreePorts 释放多个端口
func FreePorts(ports []int) {
	for _, port := range ports {
		//fmt.Printf("正在释放端口 %s...\n", port)
		if err := FreePort(port); err != nil {
			//fmt.Printf("释放端口 %s 失败: %v\n", port, err)
		} else {
			//fmt.Printf("端口 %s 已成功释放。\n", port)
		}
	}
}

// FreePort 释放单个端口
func FreePort(port int) error {
	pid, err := FindPIDByPort(strconv.Itoa(port))
	if err != nil {
		return err
	}

	//fmt.Printf("找到占用端口 %s 的进程 PID: %s\n", port, pid)

	if err := KillProcessByPID(pid); err != nil {
		return err
	}

	return nil
}

// GetAvailablePort 自动选择一个可用的端口
func GetAvailablePort() (int, error) {
	// 使用 net.Listen 自动选择一个可用的端口
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close() // 程序结束时关闭监听

	// 获取端口号
	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}
