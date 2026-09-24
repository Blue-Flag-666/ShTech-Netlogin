# ShTech-Netlogin

上海科技大学校园网自动登录工具，支持 Windows、Linux 和 macOS。需要验证码时由本机模型识别，无需浏览器、Python 或额外动态库。

> 非官方工具。请只使用自己的校园网账号，并遵守学校网络使用规定。

## 开始使用

1. 从 [最新 Release](https://github.com/Blue-Flag-666/ShTech-Netlogin/releases/latest) 下载适合系统的文件：

   | 系统 | x64 / Intel | ARM64 | 其他 |
   | --- | --- | --- | --- |
   | Windows | `shtu-net-login-windows-amd64.exe` | `shtu-net-login-windows-arm64.exe` | — |
   | Linux | `shtu-net-login-linux-amd64` | `shtu-net-login-linux-arm64` | ARMv7：`shtu-net-login-linux-armv7` |
   | macOS | `shtu-net-login-darwin-amd64` | `shtu-net-login-darwin-arm64` | — |

   为方便运行，可将文件改名为 `shtu-net-login`（Windows 为 `shtu-net-login.exe`），放到系统的 `PATH` 目录中。Linux/macOS 下载后还需运行 `chmod +x shtu-net-login`。每个下载文件都有同名的 `.sha256` 校验文件。

2. 运行 `shtu-net-login status`。首次运行会自动创建配置文件，并显示其位置：

   | 系统 | 默认配置位置 |
   | --- | --- |
   | Windows | `%AppData%\shtu-net-login\config.json` |
   | Linux | `~/.config/shtu-net-login/config.json`，或 `$XDG_CONFIG_HOME/shtu-net-login/config.json` |
   | macOS | `~/Library/Application Support/shtu-net-login/config.json` |

3. 打开配置文件，填写 `username`（学号）和 `password`。配置文件不存在时会创建空白账号密码；不会覆盖已有文件。

4. 运行 `shtu-net-login login` 登录。需要持续监测和掉线重连时，运行 `shtu-net-login watch`；按 `Ctrl+C` 停止。

如果文件不在 `PATH` 中，在 Windows PowerShell 使用 `& "文件完整路径" login`；在 Linux/macOS 使用 `./shtu-net-login login`。

## 命令

```text
shtu-net-login status         查看网络状态
shtu-net-login login          检测网络，需要认证时登录（默认命令）
shtu-net-login watch          持续监测，掉线后自动登录
shtu-net-login sample-config  输出配置示例
shtu-net-login version        输出版本
```

`online` 表示网络已连通；`captive` 表示发现认证门户。检测失败时会输出错误信息。

## 常用设置

配置文件为 JSON。只填写账号密码即可使用；其他设置可按需添加：

```json
{
  "username": "你的学号",
  "password": "你的密码",
  "fast_login": true,
  "interval": "30s"
}
```

默认先尝试一次无验证码登录；若门户不接受，就改用本地验证码识别。密码错误或账号锁定会立即停止，避免重复提交。将 `fast_login` 设为 `false` 可跳过首次快速尝试。

如果 VPN、容器或多块网卡使自动选址出错，可指定校园网 `10.x` 地址或网卡。例如：

```sh
shtu-net-login login --ip 10.19.123.45
shtu-net-login watch --interface ens192
```

显式 `ip` 优先于 `interface`；两者都未设置时，先使用门户返回的地址，再自动查找本机 `10.x` 地址。选项须写在命令之后。

| 选项 | 用途 |
| --- | --- |
| `--config PATH` | 使用指定配置文件 |
| `--ip ADDRESS` / `--interface NAME` | 指定校园网地址或网卡 |
| `--interval 1m` | 调整 `watch` 检查间隔 |
| `--timeout 15s` | 调整单次请求超时 |
| `--no-fast-login` | 直接使用验证码流程 |
| `--base-url URL` | 覆盖门户地址 |
| `--insecure` | 临时跳过 TLS 证书校验，仅在确认门户证书异常时使用 |

也可以使用环境变量覆盖配置文件：`SHTU_USERNAME`、`SHTU_PASSWORD`、`SHTU_IP`、`SHTU_INTERFACE`、`SHTU_FAST_LOGIN`、`SHTU_INTERVAL`、`SHTU_TIMEOUT`、`SHTU_BASE_URL` 和 `SHTU_INSECURE`。账号密码还兼容 `EGATE_ID`、`EGATE_PASSWORD`。不要把密码放入命令行参数或多人可读的启动脚本。

## 开机自动运行

先在终端确认 `watch` 能正常工作，再把 `shtu-net-login watch` 加入系统启动项。启动项无需保存密码；程序会读取用户配置文件。

- Windows：使用任务计划程序，在登录时运行 `shtu-net-login.exe watch`。
- Linux：使用 systemd 用户服务，设置 `ExecStart` 为程序的绝对路径加 `watch`，然后运行 `systemctl --user enable --now 服务名`。
- macOS：使用 LaunchAgent，将程序绝对路径与 `watch` 写入 `ProgramArguments`，并设置 `RunAtLoad`。

密码错误或账号锁定时，`watch` 会退出，不会持续重试。开机运行无效时，先在终端前台运行 `watch` 查看错误。

## 从源码构建

需要 Go 1.27.1 或更新版本：

```sh
go build ./cmd/shtu-net-login
```

项目的测试与交叉编译由 GitHub Actions 执行。第三方许可见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
