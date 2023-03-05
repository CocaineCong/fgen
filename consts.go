package main

const modsTemplate = `module {module}

go {version}`

const configYamlTemplate = `
system:
  domain: {domain}
  version: 1.0
  appEnv: "test"
  HttpPort: ":4000"
  Host: "localhost"

mysql:
  default:
    dialect: "mysql"
    host: "127.0.0.1"
    port: "3306"
    dbName: "hello_story"
    user: "root"
    password: "root"
    charset: "utf8mb4"

redis:
  name: 1
  address: 127.0.0.1:6379
  password: 
`

const configGolangTemplate = `
package config

import (
	"os"

	"github.com/spf13/viper"
)

var Config *Conf

type Conf struct {
	System        *System                 'yaml:"system"'
	MySql         map[string]*MySql       'yaml:"mysql"'
	Redis         *Redis                  'yaml:"redis"'
}

type MySql struct {
	Dialect  string 'yaml:"dialect"'
	DbHost   string 'yaml:"dbHost"'
	DbPort   string 'yaml:"dbPort"'
	DbName   string 'yaml:"dbName"'
	UserName string 'yaml:"userName"'
	Password string 'yaml:"password"'
	Charset  string 'yaml:"charset"'
}

type Redis struct {
	Host     string 'yaml:"redisHost"'
	Port     string 'yaml:"redisPort"'
	Password string 'yaml:"redisPwd"'
	DbName   int    'yaml:"redisDbName"'
	Network  string 'yaml:"redisNetwork"'
}

type System struct {
	AppEnv   string 'yaml:"appEnv"'
	Domain   string 'yaml:"domain"'
	Version  string 'yaml:"version"'
	HttpPort string 'yaml:"httpPort"'
	Host     string 'yaml:"host"'
}

func InitConfig() {
	workDir, _ := os.Getwd()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(workDir + "/config/local")
	viper.AddConfigPath(workDir)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&Config)
	if err != nil {
		panic(err)
	}
}
`

const cmdTemplate = `
package main

import (
	"{configPath}/config"
	"{routerPath}/router"
)

func main() {
	config.InitConfig()
	r := router.NewRouter()
	_ = r.Run(config.Config.System.HttpPort)
}
`

const routerTemplate = `
package router

import (
	"{module}/api"
	"{module}/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	ginRouter := gin.Default()
	ginRouter.Use(middleware.Cors())
	store := cookie.NewStore([]byte("something-very-secret"))
	ginRouter.Use(sessions.Sessions("mysession", store))
	v1 := ginRouter.Group("/api/v1")
	{
		v1.GET("ping", func(context *gin.Context) {
			context.JSON(200, "success")
		})
	}
	return ginRouter
}
`
const middlewareCorsTemplate = `
package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 跨域
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method               //请求方法
		origin := c.Request.Header.Get("Origin") //请求头部
		var headerKeys []string                  // 声明请求头keys
		for k := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Origin", "*")                                       // 这是允许访问所有域
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE") //服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
			//  header的类型
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			// 允许跨域设置                                                                                                      可以返回其他子段
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar") // 跨域关键设置 让浏览器可以解析
			c.Header("Access-Control-Max-Age", "172800")                                                                                                                                                           // 缓存请求信息 单位为秒
			c.Header("Access-Control-Allow-Credentials", "false")                                                                                                                                                  //  跨域请求是否需要带cookie信息 默认设置为true
			c.Set("content-type", "application/json")                                                                                                                                                              // 设置返回格式是json
		}
		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "Options Request!")
		}
		// 处理请求
		c.Next() //  处理请求
	}
}
`

const ScriptCmdTemplate = `
cd {path}
go env -w GOPROXY=https://goproxy.cn,direct
go mod tidy
`
