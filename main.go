package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"
)

const (
	DefaultGenModelPath = "./repository/dao/"        // 默认生成model的路径
	DefaultConfigPath   = "config/local/config.yaml" // 默认的配置文件路径
	DefaultKey          = "default"                  // 默认的mysql的key
	Version             = "0.0.1"                    // 版本号
)

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "project",
			Usage: "fgen project",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name",
					Usage: "set the project name",
				},
				cli.StringFlag{
					Name:  "group",
					Usage: "set the project group",
				},
				cli.StringFlag{
					Name:  "b",
					Usage: "set download the project branch",
				},
			},
			Action: func(ctx *cli.Context) error {
				name := ctx.String("name")
				if name == "" {
					return errors.New("must set project name")
				}

				group := ctx.String("name")
				if group == "" {
					group = "sns"
				}

				// b := ctx.String("b")
				return GenProject("", name)
			},
		},
		{
			Name:   "model",
			Usage:  "gen table model",
			Flags:  modelFlag(),
			Action: modelAction(),
		},
		{
			Name:      "version",
			ShortName: "v",
			Usage:     "gen version",
			Action: func(ctx *cli.Context) {
				fmt.Println(Version)
				return
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
	}
}

func modelFlag() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "dns",
			Usage: "depreacted, use -dsn instead",
		},
		cli.StringFlag{
			Name:  "dsn", // Data Source Name
			Usage: "mysql link dsn , default read config/local/config.yaml",
		},
		cli.StringFlag{
			Name:  "t",
			Usage: "gen the tables name, separable use ,",
		},
		cli.StringFlag{
			Name:  "p",
			Usage: "model generation path",
		},
		cli.StringFlag{
			Name:  "c",
			Usage: "config.yaml path",
		},
		cli.StringFlag{
			Name:  "k",
			Usage: "mysql config key",
		},
	}
}

func modelAction() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		dsn := ctx.String("dsn")
		if dsn == "" {
			dsn = ctx.String("dns")
		}
		t := ctx.String("t")
		configPath := ctx.String("c")
		path := ctx.String("p")
		key := ctx.String("k")

		if t == "" {
			return fmt.Errorf("the table name must be specified")
		}

		if path == "" {
			path = DefaultGenModelPath
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				path = DefaultGenModelPath
			}
		}

		if configPath == "" {
			configPath = DefaultConfigPath
		}

		if key == "" {
			key = DefaultKey
		}

		tables := strings.Split(t, ",")
		return GenModel(context.Background(), dsn, path, "", configPath, key, tables...)
	}
}
