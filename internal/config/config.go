package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://net-auth.shanghaitech.edu.cn:19008"

var DefaultProbes = []Probe{
	{URL: "http://connectivitycheck.gstatic.com/generate_204", Status: 204},
	{URL: "http://captive.apple.com/hotspot-detect.html", Status: 200, Contains: "Success"},
	{URL: "http://www.msftconnecttest.com/connecttest.txt", Status: 200, Contains: "Microsoft Connect Test"},
}

type Probe struct {
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Contains string `json:"contains,omitempty"`
}

type fileConfig struct {
	Username           string  `json:"username"`
	Password           string  `json:"password"`
	BaseURL            string  `json:"base_url"`
	Interval           string  `json:"interval"`
	Timeout            string  `json:"timeout"`
	MaxCaptchaAttempts int     `json:"max_captcha_attempts"`
	Insecure           bool    `json:"insecure"`
	Probes             []Probe `json:"probes"`
}

type Config struct {
	Username           string
	Password           string
	BaseURL            string
	Interval           time.Duration
	Timeout            time.Duration
	MaxCaptchaAttempts int
	Insecure           bool
	Probes             []Probe
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "shtu-net-login", "config.json"), nil
}

func Load(path string) (Config, error) {
	cfg := Config{
		BaseURL:            DefaultBaseURL,
		Interval:           30 * time.Second,
		Timeout:            12 * time.Second,
		MaxCaptchaAttempts: 5,
		Probes:             append([]Probe(nil), DefaultProbes...),
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("读取配置文件: %w", err)
		}
		if err == nil {
			var f fileConfig
			if err := json.Unmarshal(data, &f); err != nil {
				return Config{}, fmt.Errorf("解析配置文件: %w", err)
			}
			cfg.Username = f.Username
			cfg.Password = f.Password
			if f.BaseURL != "" {
				cfg.BaseURL = f.BaseURL
			}
			if f.Interval != "" {
				cfg.Interval, err = time.ParseDuration(f.Interval)
				if err != nil {
					return Config{}, fmt.Errorf("interval: %w", err)
				}
			}
			if f.Timeout != "" {
				cfg.Timeout, err = time.ParseDuration(f.Timeout)
				if err != nil {
					return Config{}, fmt.Errorf("timeout: %w", err)
				}
			}
			if f.MaxCaptchaAttempts > 0 {
				cfg.MaxCaptchaAttempts = f.MaxCaptchaAttempts
			}
			cfg.Insecure = f.Insecure
			if len(f.Probes) > 0 {
				cfg.Probes = f.Probes
			}
		}
	}

	firstEnv(&cfg.Username, "SHTU_USERNAME", "EGATE_ID")
	firstEnv(&cfg.Password, "SHTU_PASSWORD", "EGATE_PASSWORD")
	firstEnv(&cfg.BaseURL, "SHTU_BASE_URL")
	if value := os.Getenv("SHTU_INTERVAL"); value != "" {
		var err error
		cfg.Interval, err = time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("SHTU_INTERVAL: %w", err)
		}
	}
	if value := os.Getenv("SHTU_TIMEOUT"); value != "" {
		var err error
		cfg.Timeout, err = time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("SHTU_TIMEOUT: %w", err)
		}
	}
	if value := os.Getenv("SHTU_INSECURE"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("SHTU_INSECURE: %w", err)
		}
		cfg.Insecure = parsed
	}

	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Interval <= 0 || cfg.Timeout <= 0 || cfg.MaxCaptchaAttempts <= 0 {
		return Config{}, errors.New("interval、timeout 和 max_captcha_attempts 必须大于 0")
	}
	return cfg, nil
}

func (c Config) ValidateCredentials() error {
	if c.Username == "" || c.Password == "" {
		return errors.New("未配置账号密码：请设置 SHTU_USERNAME/SHTU_PASSWORD，或填写配置文件")
	}
	return nil
}

func firstEnv(target *string, names ...string) {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			*target = value
			return
		}
	}
}

func Sample() string {
	return `{
  "username": "你的学号",
  "password": "你的密码",
  "interval": "30s",
  "timeout": "12s",
  "max_captcha_attempts": 5,
  "insecure": false
}`
}
