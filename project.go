package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

func GenProject(projectPath, projectName string) error {

	if projectPath == "" {
		projectPath = "./"
	}

	if projectName == "" {
		// 外层能保证不为空
		projectName = "demo/"
	}

	genApiPath := projectPath + projectName + "api/"
	genCmdPath := projectPath + projectName + "cmd/"
	genConfigPath := projectPath + projectName + "config/"
	genConstsPath := projectPath + projectName + "consts/"
	genLoadingPath := projectPath + projectName + "loading/"
	genMiddlewarePath := projectPath + projectName + "middleware/"
	genPkgPath := projectPath + projectName + "pkg/"
	genRepositoryPath := projectPath + projectName + "repository/"
	genRouterPath := projectPath + projectName + "router/"
	genSerializerPath := projectPath + projectName + "serializer/"
	genServicePath := projectPath + projectName + "service/"
	genTestPath := projectPath + projectName + "test/"
	genTypesPath := projectPath + projectName + "types/"
	genModsPath := projectPath + projectName + "go.mod"

	genPaths := []string{genApiPath, genCmdPath, genConfigPath, genConstsPath, genLoadingPath, genMiddlewarePath, genPkgPath, genRepositoryPath, genRouterPath, genSerializerPath, genServicePath, genTestPath, genTypesPath, genModsPath}

	for _, genPath := range genPaths {

		switch genPath {
		case genModsPath:
			version, _ := getGolangVersion()
			entityContent := gstr.ReplaceByMap(modsTemplate, g.MapStrStr{
				"{module}":  projectName[:len(projectName)-1],
				"{version}": version,
			})
			if err := writeFile(genModsPath, entityContent); err != nil {
				return err
			}

		case genConfigPath:
			if err := gfile.Mkdir(genConfigPath); err != nil {
				glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
			}
			if err := gfile.Mkdir(genConfigPath + "local/"); err != nil {
				glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
			}
			yamlPath := genConfigPath + "local/config.yaml"
			entityContent := gstr.ReplaceByMap(configYamlTemplate, g.MapStrStr{
				"{domain}": projectName[:len(projectName)-1],
			})

			if err := writeFile(yamlPath, entityContent); err != nil {
				return err
			}

			entityContent = strings.Replace(configGolangTemplate, "'", "`", -1)
			configGo := genConfigPath + "config.go"
			if err := writeFile(configGo, entityContent); err != nil {
				return err
			}

		case genCmdPath:
			if err := gfile.Mkdir(genCmdPath); err != nil {
				glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
			}
			cmdPath := genCmdPath + "main.go"
			entityContent := gstr.ReplaceByMap(cmdTemplate, g.MapStrStr{
				"{configPath}": projectName[:len(projectName)-1],
				"{routerPath}": projectName[:len(projectName)-1],
			})
			if err := writeFile(cmdPath, entityContent); err != nil {
				return err
			}

		case genRouterPath:
			if err := gfile.Mkdir(genRouterPath); err != nil {
				glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
			}
			routerPath := genRouterPath + "router.go"
			entityContent := gstr.ReplaceByMap(routerTemplate, g.MapStrStr{
				"{module}": projectName[:len(projectName)-1],
			})
			if err := writeFile(routerPath, entityContent); err != nil {
				return err
			}

		case genMiddlewarePath:
			if err := gfile.Mkdir(genMiddlewarePath); err != nil {
				glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
			}
			middlewarePath := genMiddlewarePath + "cors.go"
			entityContent := gstr.ReplaceByMap(middlewareCorsTemplate, g.MapStrStr{
				"{module}": projectName[:len(projectName)-1],
			})
			if err := writeFile(middlewarePath, entityContent); err != nil {
				return err
			}

		default:
			if err := gfile.Mkdir(genPath); err != nil {
				glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
			}
		}
	}

	pingCmd := exec.Command("ping", "-c 5 baidu.com")
	_, err := pingCmd.CombinedOutput()
	if err != nil {
		fmt.Println("err", err)
		return err
	}

	entityContent := gstr.ReplaceByMap(ScriptCmdTemplate, g.MapStrStr{
		"{path}": projectPath + projectName[:len(projectName)-1],
	})
	scriptPath := projectPath + "start.sh"
	if err := writeFile(scriptPath, entityContent); err != nil {
		return err
	}
	cmd := exec.Command("bash", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("err", err)
		return err
	}
	fmt.Println("output", string(output))
	return nil
}

func getGolangVersion() (string, error) {
	ver := runtime.Version()
	if ver != "" {
		return ver[2:6], nil
	}
	return "", errors.New("golang 环境不存在")
}

func writeFile(path, content string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	defer file.Close()
	// 写入文件时，使用带缓存的 *Writer
	write := bufio.NewWriter(file)
	_, err = write.WriteString(content)
	if err != nil {
		return err
	}
	err = write.Flush()

	return err
}
