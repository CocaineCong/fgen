package main

import (
	"fmt"
	"testing"
)

func TestGenProject(t *testing.T) {
	err := GenProject("")
	if err != nil {
		fmt.Println("err", err)
	}
	fmt.Println("创建成功")
}
