package main

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestGenProject(t *testing.T) {
	projectPath := "/Users/mac/GolandProjects/genTest/"
	err := GenProject(projectPath, "")
	if err != nil {
		fmt.Println("err", err)
	}
	fmt.Println("创建成功")
}

func TestCmd(t *testing.T) {
	cmd := exec.Command("bash", "start.sh")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("err", err)
	}
	fmt.Println("output", string(output))
}
