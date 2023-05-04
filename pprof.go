package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/util/gconv"
	excelize "github.com/xuri/excelize/v2"
)

type Result struct {
	GoInfo                []*GoInfo     `sheet:"Sheet1"`
	GoWaitInfo            []*GoWaitInfo `sheet:"Go阻塞信息"`
	HeapFlatCum           []*FlatCum    `sheet:"内存占比"`
	GoFlatCum             []*FlatCum    `sheet:"Go占比"`
	CpuFlatCum            []*FlatCum    `sheet:"CPU占比"`
	HeapComparisonFlatCum []*FlatCum    `sheet:"内存打点比较"`
	GoComparisonFlatCum   []*FlatCum    `sheet:"Go打点比较"`
	CpuComparisonFlatCum  []*FlatCum    `sheet:"CPU打点比较"`
}

type GoInfo struct {
	GoTotal int `xlsx:"A-协程总数"`
}

type GoWaitInfo struct {
	GoId     int    `xlsx:"A-协程ID"`
	WaitTime int    `xlsx:"B-等待时长"`
	Reason   string `xlsx:"C-阻塞原因"`
	Stack    string `xlsx:"D-栈信息"`
}

type FlatCum struct {
	Flat           string `xlsx:"A-flat"`  // 函数自身内存分配大小
	FlatPercentage string `xlsx:"B-flat%"` // 函数自身内存与总内存占比
	SumPercentage  string `xlsx:"C-sum"`   // 每一行的flat% 与上面所有行 flat% 总和
	Cum            string `xlsx:"D-cum"`   // 当前函数加上它所有调用栈的内存大小
	CumPercentage  string `xlsx:"E-cum%"`  // 当前函数加上它所有调用栈的内存与总内存占比
	Code           string `xlsx:"F-code"`
}

const (
	defaultLine     = 10       // 默认获取行数
	defaultSeconds  = 30       // 默认采集时长
	defaultWaitTime = 10       // 两次打点间隔
	defaultAppName  = "fanone" // 默认AppName
	cpu             = "profile"
	goroutine       = "goroutine"
	heap            = "heap"
)

var (
	defaultOptions = []string{"heap", "goroutine", "profile"}
	notOpenPprof   = errors.New("pprof not open")
	logger         = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)
)

func GenReport(ip string, seconds int, line int, options ...string) error {
	if seconds == 0 {
		seconds = defaultSeconds
	}

	if line == 0 {
		line = defaultLine
	}

	if len(options) == 0 {
		options = defaultOptions
	}

	// 获取appName ，并判断是否开启了pprof
	appName, err := GetAppName(ip)
	if err != nil {
		return err
	}

	var (
		result = &Result{}
		wg     sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		defer wg.Done()
		err := DownloadTrace(ip, appName, seconds)
		if err != nil {
			logger.Println("recover", err)
		}
	}()

	for _, option := range options {
		wg.Add(1)
		go func(option string) {
			defer func() {
				if err := recover(); err != nil {
					logger.Println("recover", err)
				}
			}()
			defer wg.Done()

			fcs, err := GetFlatAndCum(ip, option, line)
			if err != nil {
				logger.Println(err)
			}

			cs, err := Comparison(ip, option, defaultWaitTime)
			if err != nil {
				logger.Println(err)
			}

			switch option {
			case cpu:
				result.CpuFlatCum = fcs
				result.CpuComparisonFlatCum = cs
				err := DownloadFireCPU(ip, appName, seconds)
				if err != nil {
					logger.Println("下载CPU火焰图失败", err)
				}
			case goroutine:
				result.GoFlatCum = fcs
				result.GoComparisonFlatCum = cs

				goWaitInfos, err := GoroutineMaxWaitTime(ip)
				if err != nil {
					logger.Println("获取goroutine阻塞信息", err)
				}
				result.GoWaitInfo = goWaitInfos

				goTotal, err := GoroutineNum(ip)
				if err != nil {
					logger.Println("获取goroutine总数失败", err)
				}
				result.GoInfo = []*GoInfo{
					{
						GoTotal: goTotal,
					},
				}

			case heap:
				result.HeapFlatCum = fcs
				result.HeapComparisonFlatCum = cs

				if err = DownloadHeap(ip, appName); err != nil {
					logger.Println("下载内存分析图失败", err)
				}

				if err = DownloadFireHeap(ip, appName); err != nil {
					logger.Println("下载内存火焰图失败", err)
				}
			}
		}(option)
	}

	wg.Wait()

	return exportExcel(appName, result)
}

func GetFlatAndCum(ip, option string, n int) ([]*FlatCum, error) {
	fmt.Printf("获取%s占比...\n", option)
	url := fmt.Sprintf("http://%s/debug/pprof/%s", ip, option)
	cmd := exec.Command("go", "tool", "pprof", url)
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	_, err = pipe.Write(bytes.NewBufferString(fmt.Sprintf("top %d", n)).Bytes())
	if err != nil {
		return nil, err
	}
	_ = pipe.Close()
	result, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseFlatCum(result, n)
}

func GetAppName(ip string) (string, error) {
	var appName string
	url := fmt.Sprintf("http://%s/debug/pprof/cmdline", ip)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return defaultAppName, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return defaultAppName, notOpenPprof
	}

	bs, _ := ioutil.ReadAll(resp.Body)
	appName = string(bs)
	appName = strings.ReplaceAll(appName, "/app/", "")
	return appName, nil
}

func GoroutineNum(ip string) (int, error) {
	url := fmt.Sprintf("http://%s/dubug/pprof/goroutine?debug=1", ip)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	br := bufio.NewReader(resp.Body)
	line, _, _ := br.ReadLine()
	re := regexp.MustCompile(`goroutine profile: total ([0-9]*)`)

	matchs := re.FindSubmatch(line)
	if len(matchs) >= 1 {
		return gconv.Int(string(matchs[1])), nil
	}
	return 0, errors.New("not found")
}

// 阻塞最长的goroutine
// 按阻塞时长倒叙排列
func GoroutineMaxWaitTime(ip string) ([]*GoWaitInfo, error) {
	url := fmt.Sprintf("http://%s/debug/pprof/goroutine?debug=2", ip)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bs, _ := ioutil.ReadAll(resp.Body)

	stacks := strings.Split(string(bs), "\n\n")

	re := regexp.MustCompile(`goroutine\s([0-9]*\s\[(.*),\s([0-9]*)\sminutes\])`)
	var infos []*GoWaitInfo
	for _, s := range stacks {
		match := re.FindStringSubmatch(s)

		if len(match) > 2 {
			info := &GoWaitInfo{
				GoId:     gconv.Int(match[1]),
				Reason:   match[2],
				WaitTime: gconv.Int(match[3]),
				Stack:    s,
			}
			infos = append(infos, info)
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].WaitTime > infos[j].WaitTime
	})

	return infos, nil
}

func DownloadTrace(ip, appName string, second int) error {
	fmt.Println("download trace.out...")

	url := fmt.Sprintf("http://%s/debug/pprof/trace?seconds=%d", ip, second)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	fileName := fmt.Sprintf("%s_trace.out", appName)
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return err

}

// 解析flat与cum
// length 解析top几
func parseFlatCum(result []byte, length int) ([]*FlatCum, error) {
	bs := bytes.NewBuffer(result)
	br := bufio.NewReader(bs)
	var (
		flag          bool
		flatCums      []*FlatCum
		currentLength int
	)

	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}
		if flag {
			infos := make([]string, 0)
			ss := strings.Split(string(line), " ")
			for _, s := range ss {
				s = strings.ReplaceAll(s, " ", "")
				if s == "" {
					continue
				}
				infos = append(infos, s)
			}
			if len(infos) < 6 {
				break
			}
			if currentLength >= length {
				break
			}

			flatCum := &FlatCum{
				Flat:           infos[0],
				FlatPercentage: infos[1],
				SumPercentage:  infos[2],
				Cum:            infos[3],
				CumPercentage:  infos[4],
				Code:           infos[5],
			}
			flatCums = append(flatCums, flatCum)
			currentLength++
		}
		if !flag {
			if strings.Contains(string(line), "flat") {
				flag = true
			}
		}
	}
	return flatCums, nil
}

// 下载内存占比关系调用图
// --inuse space 常驻内存占比情况
// --alloc objects 临时内存分配情况
func DownloadHeap(ip, appName string) error {
	wg := new(sync.WaitGroup)
	url := fmt.Sprintf("http://%s/debug/pprof/heap", ip)
	wg.Add(1)

	go func() {
		fmt.Println("download heap alloc_space...")
		defer wg.Done()

		cmd := exec.Command("go", "tool", "pprof", "-alloc_space", "-cum", "-svg", url)

		bs, err := cmd.Output()
		if err != nil {
			logger.Println("download heap", err)
		}

		err = ioutil.WriteFile(appName+"_alloc_space.svg", bs, os.ModePerm)

		if err != nil {
			logger.Println(err)
		}

	}()

	wg.Add(1)

	go func() {
		fmt.Println("download heap inuse_space...")
		defer wg.Done()

		cmd := exec.Command("go", "tool", "pprof", "-inuse_space", "-cum", "-svg", url)
		bs, err := cmd.Output()
		if err != nil {
			logger.Println(err)
		}

		err = ioutil.WriteFile(appName+"_inuse_space.svg", bs, os.ModePerm)
		if err != nil {
			logger.Println(err)
		}
	}()

	wg.Wait()

	return nil
}

// 下载堆内存占比火焰图
func DownloadFireHeap(ip, appName string) error {
	err := exec.Command("which", "go-torch").Run()
	if err != nil {
		return errors.New("go-torch 未安装")
	}
	// 下载内存临时分配火焰图
	fmt.Println("download heap alloc_space_fire...")

	url := fmt.Sprintf("http://%s/debug/pprof/heap", ip)
	cmd := exec.Command("go-torch", "-alloc_space", url, "--colors=mem")

	err = cmd.Run()
	if err != nil {
		return err
	}

	err = os.Rename("torch.svg", appName+"_alloc_space_fire.svg")

	if err != nil {
		return err
	}

	// 下载正在使用的内存分配火焰图
	fmt.Println("download heap inuse_space_fire...")
	cmd = exec.Command("go-torch", "-inuse_space", url, "--colors=-mem")
	err = cmd.Run()
	if err != nil {
		return err
	}

	err = os.Rename("torch.svg", appName+"_inuse_space_fire.svg")
	if err != nil {
		return err
	}

	return nil
}

// 下载CPU耗时火焰图
func DownloadFireCPU(ip, appName string, seconds int) error {
	err := exec.Command("which", "go-torch").Run()
	if err != nil {
		return errors.New("go-torch 未安装")
	}

	fmt.Println("download profile fire...")
	url := fmt.Sprintf("http://%s/debug/pprof/profile", ip)
	cmd := exec.Command("go-torch", "--seconds", fmt.Sprintf("%d", seconds), url)

	if err = cmd.Run(); err != nil {
		return err
	}

	if err = os.Rename("torch.svg", appName+"_profile.svg"); err != nil {
		return err
	}

	return nil
}

// 打点比较
// option：goroutine、heaps、profile
// waitTime：两次打点差异
func Comparison(ip, option string, waitTime int) ([]*FlatCum, error) {
	fmt.Printf("打点比较%s...\n\n", option)
	url := fmt.Sprintf("http://%s/debug/pprof/%s", ip, option)
	// 打第一个点
	f1 := timerInfo(url, "001")

	// 等待一会，打第二个点
	time.Sleep(time.Duration(waitTime) * time.Second)
	f2 := timerInfo(url, "002")

	// 清理
	defer func() {
		_ = os.Remove(f1)
		_ = os.Remove(f2)
	}()

	// 分析报告
	return parseComparison(f1, f2)
}

func parseComparison(f1, f2 string) ([]*FlatCum, error) {
	cmd := exec.Command("go", "tool", "pprof", "-base", f1, f2)
	i, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	_, err = i.Write([]byte("top"))
	if err != nil {
		logger.Println(err)
	}
	_ = i.Close()

	result, err := cmd.CombinedOutput()
	if err != nil {
		logger.Println(err)
	}

	return parseFlatCum(result, 10)
}

func timerInfo(url, number string) string {
	cmd := exec.Command("go", "tool", "pprof", url)
	bs, err := cmd.CombinedOutput()
	if err != nil {
		logger.Println(err)
	}
	re := regexp.MustCompile(`Saved profile in \s(.*)`)
	result := re.FindStringSubmatch(string(bs))

	var (
		saveFilePath string
		fileName     string
	)

	if len(result) > 1 {
		saveFilePath = result[1]
		ps := strings.Split(saveFilePath, "/")
		fileName = ps[len(ps)-1]
		if ts := strings.Split(fileName, "."); len(ts) > 5 {
			ts[3] = number
			fileName = strings.Join(ts, ".")
		}
		err = os.Rename(saveFilePath, fileName)
		if err != nil {
			logger.Println(err)
		}
	}
	return fileName
}

func exportExcel(appName string, result *Result) error {
	var (
		err  error
		xlsx = excelize.NewFile()
	)

	ve := reflect.ValueOf(result).Elem()
	te := reflect.TypeOf(result).Elem()

	for j := 0; j < ve.NumField(); j++ {
		vf := ve.Field(j)
		tf := te.Field(j)
		v := reflect.ValueOf(vf.Interface())
		if v.IsNil() {
			continue
		}
		sheetName := tf.Tag.Get("sheet")
		xlsx, err = WriteXlsx(xlsx, sheetName, gconv.Interfaces(vf.Interface()))
		if err != nil {
			logger.Println(err)
		}
	}
	return xlsx.SaveAs(appName + ".xlsx")
}
