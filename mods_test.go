package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestGenProject(t *testing.T) {
	projectPath, _ := os.Getwd()
	err := GenProject(projectPath, "demo/")
	if err != nil {
		fmt.Println("err", err)
	}
	fmt.Println("创建成功")
}

func TestCmd(t *testing.T) {
	// cmd := exec.Command("bash", "start.sh")
	cmd := exec.Command("ping", "baidu.com -c 5 ")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("err", err)
	}
	fmt.Println("output", string(output))
}

func TestPingBaidu(t *testing.T) {
	hostname := "baidu.com"
	timeout := time.Second * 5

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:80", hostname), timeout)
	if err != nil {
		fmt.Printf("Error connecting to host: %s\n", err.Error())
		return
	}

	defer conn.Close()

	fmt.Println("Connected to", conn.RemoteAddr().String())
}
