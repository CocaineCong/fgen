package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/gfile"
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/text/gstr"
)

const (
	defaultGroup = "fanone"
	defaultName  = "cocainecong"
	defaultPath  = "./fanone"
)

func GenProject(projectName string) error {
	if projectName == "" {
		// 外层能保证不为空
		projectName = "demo/"
	}

	genApiPath := projectName + "api/"
	genCmdPath := projectName + "cmd/"
	genConfigPath := projectName + "config/"
	genConstsPath := projectName + "consts/"
	genLoadingPath := projectName + "loading/"
	genMiddlewarePath := projectName + "middleware/"
	genPkgPath := projectName + "pkg/"
	genRepositoryPath := projectName + "repository/"
	genRouterPath := projectName + "router/"
	genSerializerPath := projectName + "serializer/"
	genServicePath := projectName + "service/"
	genTestPath := projectName + "test/"
	genTypesPath := projectName + "types/"
	genModsPath := projectName + "go.mod"

	genPaths := []string{genApiPath, genCmdPath, genConfigPath, genConstsPath, genLoadingPath, genMiddlewarePath, genPkgPath, genRepositoryPath, genRouterPath, genSerializerPath, genServicePath, genTestPath, genTypesPath, genModsPath}

	for _, genPath := range genPaths {

		switch genPath {
		case genModsPath:
			version, _ := getGolangVersion()
			entityContent := gstr.ReplaceByMap(modsTemplate, g.MapStrStr{
				"{module}":  projectName[:len(projectName)-1],
				"{version}": version,
			})

			file, err := os.OpenFile(genModsPath, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				fmt.Println("文件打开失败", err)
			}
			defer file.Close()
			// 写入文件时，使用带缓存的 *Writer
			write := bufio.NewWriter(file)
			_, err = write.WriteString(entityContent)
			if err != nil {
				return err
			}
			write.Flush()

		default:
			if err := gfile.Mkdir(genPath); err != nil {
				glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
			}
		}
	}
	return nil
}

func getGolangVersion() (string, error) {
	ver := runtime.Version()
	if ver != "" {
		return ver[2:6], nil
	}
	return "", errors.New("不存在")
}

const modsTemplate = `module {module}

go {version}`
