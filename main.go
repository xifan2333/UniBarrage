package main

import (
	"UniBarrage/services/api"
	"UniBarrage/services/proxy"
	ws "UniBarrage/services/websockets"
	"UniBarrage/utils/cors"
	"UniBarrage/utils/trace"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "UniBarrage",
		Usage: "启动 UniBarrage 服务",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "wsHost",
				Aliases: []string{"wh"},
				Value:   "127.0.0.1",
				Usage:   "WebSocket 主机地址",
			},
			&cli.IntFlag{
				Name:    "wsPort",
				Aliases: []string{"wp"},
				Value:   7777,
				Usage:   "WebSocket 端口",
			},
			&cli.StringFlag{
				Name:    "apiHost",
				Aliases: []string{"ah"},
				Value:   "127.0.0.1",
				Usage:   "API 主机地址",
			},
			&cli.IntFlag{
				Name:    "apiPort",
				Aliases: []string{"ap"},
				Value:   8080,
				Usage:   "API 端口",
			},
			&cli.BoolFlag{
				Name:    "useProxy",
				Aliases: []string{"up"},
				Value:   false,
				Usage:   "是否使用代理",
			},
			&cli.StringFlag{
				Name:    "proxyHost",
				Aliases: []string{"ph"},
				Value:   "127.0.0.1",
				Usage:   "代理主机地址",
			},
			&cli.IntFlag{
				Name:    "proxyPort",
				Aliases: []string{"pp"},
				Value:   8888,
				Usage:   "代理端口",
			},
			&cli.StringFlag{
				Name:    "certFile",
				Aliases: []string{"cf"},
				Usage:   "证书文件路径",
			},
			&cli.StringFlag{
				Name:    "keyFile",
				Aliases: []string{"kf"},
				Usage:   "私钥文件路径",
			},
			&cli.StringFlag{
				Name:    "allowedOrigins",
				Aliases: []string{"ao"},
				Value:   "*",
				Usage:   "允许跨域请求的来源列表 (用逗号分隔)",
			},
			&cli.StringFlag{
				Name:    "authToken",
				Aliases: []string{"at"},
				Usage:   "用于验证的 Bearer Token (仅 API)",
			},
			&cli.IntFlag{
				Name:    "logLevel",
				Aliases: []string{"ll"},
				Value:   0,
				Usage:   "日志等级 (0: 默认, 1: 简洁, 2: 静默)",
			},
		},
		Action: func(c *cli.Context) error {
			// 初始化 Trace
			trace.Init(c.Int("logLevel"))

			// 处理允许的来源列表
			origins := cors.ParseOrigins(c.String("allowedOrigins"))

			// 启动 API 服务器
			go api.StartServer(
				c.String("apiHost"),
				c.Int("apiPort"),
				c.String("certFile"),
				c.String("keyFile"),
				c.String("authToken"),
				origins,
			)

			// 启动 WebSocket 服务器
			ws.StartServer(
				c.String("wsHost"),
				c.Int("wsPort"),
				c.String("certFile"),
				c.String("keyFile"),
				origins,
			)

			// 如果启用代理，则启动代理服务器
			if c.Bool("useProxy") {
				go proxy.StartServer(
					c.String("proxyHost"),
					c.Int("proxyPort"),
					c.String("certFile"),
					c.String("keyFile"),
					origins,
				)
			}

			// 处理程序信号以进行优雅退出
			trace.HandleSignal()

			return nil
		},
	}

	_ = app.Run(os.Args)
}
