package main

const modsTemplate = `module {module}

go {version}`

const configYamlTemplate = `
server:
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
	"time"

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
	"{configPath}/loading"
	"{configPath}/router"
)

func main() {
	config.InitConfig()
	loading.Loading()
	r := router.NewRouter()
	_ = r.Run(config.Config.System.HttpPort)
}
`
