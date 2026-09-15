package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Blue-Flag-666/ShTech-Netlogin/internal/auth"
	"github.com/Blue-Flag-666/ShTech-Netlogin/internal/captcha"
	"github.com/Blue-Flag-666/ShTech-Netlogin/internal/config"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Printf("错误: %v", err)
		os.Exit(1)
	}
}

func run() error {
	command := "login"
	args := os.Args[1:]
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		command, args = args[0], args[1:]
	}
	if command == "version" {
		fmt.Println(version)
		return nil
	}
	if command == "sample-config" {
		fmt.Println(config.Sample())
		return nil
	}

	defaultPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	flags := flag.NewFlagSet(filepath.Base(os.Args[0])+" "+command, flag.ContinueOnError)
	configPath := flags.String("config", defaultPath, "配置文件路径")
	baseURL := flags.String("base-url", "", "覆盖认证门户地址")
	interval := flags.Duration("interval", 0, "watch 检查间隔，例如 30s")
	timeout := flags.Duration("timeout", 0, "单次网络请求超时")
	insecure := flags.Bool("insecure", false, "跳过门户 TLS 证书校验（不推荐）")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("无法识别的参数: %v", flags.Args())
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *baseURL != "" {
		cfg.BaseURL = *baseURL
	}
	if *interval > 0 {
		cfg.Interval = *interval
	}
	if *timeout > 0 {
		cfg.Timeout = *timeout
	}
	if *insecure {
		cfg.Insecure = true
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch command {
	case "status":
		client, err := auth.New(cfg, nil)
		if err != nil {
			return err
		}
		state, params, err := client.Check(ctx)
		if err != nil {
			return err
		}
		if state == auth.Captive {
			fmt.Printf("%s (%s)\n", state, params.IPAddress)
		} else {
			fmt.Println(state)
		}
		return nil
	case "login", "watch":
		if err := cfg.ValidateCredentials(); err != nil {
			return err
		}
		recognizer, err := captcha.New()
		if err != nil {
			return err
		}
		client, err := auth.New(cfg, recognizer)
		if err != nil {
			return err
		}
		if command == "login" {
			return loginOnce(ctx, client)
		}
		return watch(ctx, client, cfg.Interval)
	default:
		return fmt.Errorf("未知命令 %q；可用命令: login, watch, status, sample-config, version", command)
	}
}

func loginOnce(ctx context.Context, client *auth.Authenticator) error {
	state, params, err := client.Check(ctx)
	if err != nil {
		return fmt.Errorf("检测网络状态: %w", err)
	}
	if state == auth.Online {
		log.Print("网络已经连通，无需登录")
		return nil
	}
	if state != auth.Captive {
		return errors.New("当前未发现校园网认证门户")
	}
	log.Printf("发现认证门户，客户端地址 %s", params.IPAddress)
	if err := client.Login(ctx, params); err != nil {
		return err
	}
	log.Print("登录成功")
	return nil
}

func watch(ctx context.Context, client *auth.Authenticator, interval time.Duration) error {
	log.Printf("开始监测网络，检查间隔 %s", interval)
	last := auth.Offline
	for {
		state, params, err := client.Check(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if last != auth.Offline {
				log.Printf("网络检测失败: %v", err)
			}
			last = auth.Offline
		} else if state == auth.Captive {
			log.Printf("网络需要认证，正在为 %s 登录", params.IPAddress)
			if err := client.Login(ctx, params); err != nil {
				var permanent *auth.PermanentError
				if errors.As(err, &permanent) {
					return err
				}
				log.Printf("登录失败，将稍后重试: %v", err)
			} else {
				log.Print("登录成功")
				last = auth.Online
			}
		} else {
			if last != auth.Online {
				log.Print("网络已连通")
			}
			last = auth.Online
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}
