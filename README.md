# ShTech-Netlogin

上海科技大学校园网自动登录工具。验证码完全在本机识别，无需浏览器、Python、ONNX Runtime 或其他动态库。

> 本项目是非官方工具。请只使用自己的校园网账号，并遵守学校网络使用规定。

## 支持平台

| 系统 | 架构 | Release 文件 |
| --- | --- | --- |
| Windows | x64 | `shtu-net-login-windows-amd64.exe` |
| Windows | ARM64 | `shtu-net-login-windows-arm64.exe` |
| Linux | x64 | `shtu-net-login-linux-amd64` |
| Linux | ARM64 | `shtu-net-login-linux-arm64` |
| Linux | ARMv7 | `shtu-net-login-linux-armv7` |
| macOS | Intel | `shtu-net-login-darwin-amd64` |
| macOS | Apple Silicon | `shtu-net-login-darwin-arm64` |

从 [Releases](https://github.com/Blue-Flag-666/ShTech-Netlogin/releases/latest) 下载对应文件。每个二进制旁边都有 `.sha256` 校验文件。

## Windows 使用

以下命令在 PowerShell 中运行。它们会自动选择 x64 或 ARM64 版本，并安装到当前用户目录：

```powershell
$installDir = Join-Path $env:LOCALAPPDATA "Programs\ShTech-Netlogin"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
$asset = if ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq [Runtime.InteropServices.Architecture]::Arm64) { "shtu-net-login-windows-arm64.exe" } else { "shtu-net-login-windows-amd64.exe" }
$exe = Join-Path $installDir "shtu-net-login.exe"
Invoke-WebRequest "https://github.com/Blue-Flag-666/ShTech-Netlogin/releases/latest/download/$asset" -OutFile $exe
& $exe version
```

创建配置文件并用记事本填写学号和密码：

```powershell
$configDir = Join-Path $env:APPDATA "shtu-net-login"
$configPath = Join-Path $configDir "config.json"
New-Item -ItemType Directory -Force -Path $configDir | Out-Null
[IO.File]::WriteAllText($configPath, ((& $exe sample-config) -join "`n"), [Text.UTF8Encoding]::new($false))
notepad $configPath
```

保存后测试：

```powershell
& $exe status
& $exe login
& $exe watch
```

`watch` 会持续运行；按 `Ctrl+C` 停止。

## Linux 使用

下载安装；脚本会根据当前机器选择 x64、ARM64 或 ARMv7：

```sh
case "$(uname -m)" in
  x86_64) asset=shtu-net-login-linux-amd64 ;;
  aarch64|arm64) asset=shtu-net-login-linux-arm64 ;;
  armv7l|armv7*) asset=shtu-net-login-linux-armv7 ;;
  *) echo "不支持的架构: $(uname -m)" >&2; exit 1 ;;
esac
install -d "$HOME/.local/bin"
curl -fL "https://github.com/Blue-Flag-666/ShTech-Netlogin/releases/latest/download/$asset" -o "$HOME/.local/bin/shtu-net-login"
chmod 755 "$HOME/.local/bin/shtu-net-login"
"$HOME/.local/bin/shtu-net-login" version
```

创建配置文件并编辑：

```sh
config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/shtu-net-login"
mkdir -p "$config_dir"
"$HOME/.local/bin/shtu-net-login" sample-config > "$config_dir/config.json"
chmod 600 "$config_dir/config.json"
"${EDITOR:-vi}" "$config_dir/config.json"
```

保存后测试：

```sh
"$HOME/.local/bin/shtu-net-login" status
"$HOME/.local/bin/shtu-net-login" login
"$HOME/.local/bin/shtu-net-login" watch
```

## macOS 使用

下载安装；脚本会根据当前 Mac 选择 Intel 或 Apple Silicon 版本：

```sh
case "$(uname -m)" in
  x86_64) asset=shtu-net-login-darwin-amd64 ;;
  arm64) asset=shtu-net-login-darwin-arm64 ;;
  *) echo "不支持的架构: $(uname -m)" >&2; exit 1 ;;
esac
mkdir -p "$HOME/.local/bin"
curl -fL "https://github.com/Blue-Flag-666/ShTech-Netlogin/releases/latest/download/$asset" -o "$HOME/.local/bin/shtu-net-login"
chmod 755 "$HOME/.local/bin/shtu-net-login"
"$HOME/.local/bin/shtu-net-login" version
```

创建配置文件并编辑：

```sh
config_dir="$HOME/Library/Application Support/shtu-net-login"
mkdir -p "$config_dir"
"$HOME/.local/bin/shtu-net-login" sample-config > "$config_dir/config.json"
chmod 600 "$config_dir/config.json"
"${EDITOR:-vi}" "$config_dir/config.json"
```

保存后测试：

```sh
"$HOME/.local/bin/shtu-net-login" status
"$HOME/.local/bin/shtu-net-login" login
"$HOME/.local/bin/shtu-net-login" watch
```

## 配置说明

默认配置文件位置：

- Windows：`%AppData%\shtu-net-login\config.json`
- Linux：`$XDG_CONFIG_HOME/shtu-net-login/config.json`，未设置时为 `~/.config/shtu-net-login/config.json`
- macOS：`~/Library/Application Support/shtu-net-login/config.json`

首次运行 `login`、`watch` 或 `status` 时，如果配置文件不存在，程序会自动创建目录和空白配置文件，并输出文件位置。配置文件在支持权限位的系统上以仅当前用户可读写（`0600`）创建；请填写学号和密码后再运行登录命令。

配置示例：

```json
{
  "username": "你的学号",
  "password": "你的密码",
  "interval": "30s",
  "timeout": "12s",
  "max_captcha_attempts": 5,
  "fast_login": true,
  "ip": "",
  "interface": "",
  "insecure": false
}
```

多块网卡、VPN 或容器网络同时存在多个 `10.x` 地址时，可通过 `ip` 固定客户端地址，或通过 `interface` 限定网络接口。`ip` 的优先级高于 `interface`。

默认启用 `fast_login`：每次需要认证时先尝试一次不带验证码的最小登录请求，成功时跳过 OCR；如果门户不接受则自动回退到完整验证码流程。密码错误或账号锁定不会回退重试，以免增加锁号风险。可将其设为 `false` 或使用 `--no-fast-login` 禁用。

也可以使用环境变量。环境变量优先于配置文件：

```text
SHTU_USERNAME     学号
SHTU_PASSWORD     密码
SHTU_BASE_URL     门户地址，一般无需设置
SHTU_INTERVAL     检查间隔，例如 30s
SHTU_TIMEOUT      请求超时，例如 12s
SHTU_FAST_LOGIN   是否先尝试无验证码登录，true 或 false
SHTU_IP           指定校园网 10.x IPv4 地址
SHTU_INTERFACE    指定校园网网络接口名称
SHTU_INSECURE     是否跳过 TLS 校验，true 或 false
```

同时兼容 `EGATE_ID` 和 `EGATE_PASSWORD`。不要把密码直接放入命令行参数或多人可读的启动脚本。

## 命令与选项

```text
shtu-net-login login          检测网络，需要认证时登录（默认命令）
shtu-net-login watch          持续监测，掉线后自动登录
shtu-net-login status         输出 online、captive 或 offline
shtu-net-login sample-config  输出示例配置
shtu-net-login version        输出版本
```

选项必须放在命令之后：

```text
--config PATH       指定配置文件
--interval 30s      watch 检查间隔
--timeout 12s       HTTP 请求超时
--base-url URL      覆盖门户地址
--ip ADDRESS        指定校园网 10.x IPv4 地址
--interface NAME    指定校园网网络接口
--no-fast-login     禁用无验证码快速登录
--insecure          跳过 TLS 证书校验，仅在证书异常时临时使用
```

例如：

```sh
shtu-net-login watch --interval 1m --timeout 15s
shtu-net-login login --config /path/to/config.json
shtu-net-login login --interface ens192
shtu-net-login login --ip 10.19.123.45
```

地址选择优先级为：`--ip` / `SHTU_IP` / 配置文件 `ip`，然后是对应的 `interface` 设置，再是门户重定向返回的地址，最后才自动扫描本机 `10.0.0.0/8` 地址。命令行选项会覆盖环境变量和配置文件。

## 开机自动运行

先在终端中确认 `watch` 可以成功登录，再配置开机启动。启动项中不保存密码，它会读取前面创建的配置文件。

### Windows 任务计划

在 PowerShell 中运行：

```powershell
$exe = Join-Path $env:LOCALAPPDATA "Programs\ShTech-Netlogin\shtu-net-login.exe"
$action = New-ScheduledTaskAction -Execute $exe -Argument "watch"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$principal = New-ScheduledTaskPrincipal -UserId ([Security.Principal.WindowsIdentity]::GetCurrent().Name) -LogonType Interactive -RunLevel Limited
Register-ScheduledTask -TaskName "ShTech-Netlogin" -Action $action -Trigger $trigger -Principal $principal -Description "上海科技大学校园网自动登录" -Force
Start-ScheduledTask -TaskName "ShTech-Netlogin"
Get-ScheduledTask -TaskName "ShTech-Netlogin"
```

停止并删除启动项：

```powershell
Stop-ScheduledTask -TaskName "ShTech-Netlogin" -ErrorAction SilentlyContinue
Unregister-ScheduledTask -TaskName "ShTech-Netlogin" -Confirm:$false
```

### Linux systemd 用户服务

创建并启动服务：

```sh
mkdir -p "$HOME/.config/systemd/user"
cat > "$HOME/.config/systemd/user/shtech-netlogin.service" <<'EOF'
[Unit]
Description=ShanghaiTech campus network auto login
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%h/.local/bin/shtu-net-login watch

[Install]
WantedBy=default.target
EOF
systemctl --user daemon-reload
systemctl --user enable --now shtech-netlogin.service
systemctl --user status shtech-netlogin.service
```

如需在尚未登录桌面时也启动用户服务，再执行：

```sh
sudo loginctl enable-linger "$USER"
```

查看日志：

```sh
journalctl --user -u shtech-netlogin.service -f
```

停止并删除服务：

```sh
systemctl --user disable --now shtech-netlogin.service
rm "$HOME/.config/systemd/user/shtech-netlogin.service"
systemctl --user daemon-reload
```

### macOS LaunchAgent

创建并启动 LaunchAgent：

```sh
mkdir -p "$HOME/Library/LaunchAgents" "$HOME/Library/Logs"
cat > "$HOME/Library/LaunchAgents/io.github.blue-flag-666.shtech-netlogin.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.github.blue-flag-666.shtech-netlogin</string>
  <key>ProgramArguments</key>
  <array>
    <string>$HOME/.local/bin/shtu-net-login</string>
    <string>watch</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$HOME/Library/Logs/ShTech-Netlogin.log</string>
  <key>StandardErrorPath</key>
  <string>$HOME/Library/Logs/ShTech-Netlogin.log</string>
</dict>
</plist>
EOF
plutil -lint "$HOME/Library/LaunchAgents/io.github.blue-flag-666.shtech-netlogin.plist"
launchctl bootstrap "gui/$(id -u)" "$HOME/Library/LaunchAgents/io.github.blue-flag-666.shtech-netlogin.plist"
launchctl kickstart -k "gui/$(id -u)/io.github.blue-flag-666.shtech-netlogin"
```

查看日志：

```sh
tail -f "$HOME/Library/Logs/ShTech-Netlogin.log"
```

停止并删除启动项：

```sh
launchctl bootout "gui/$(id -u)" "$HOME/Library/LaunchAgents/io.github.blue-flag-666.shtech-netlogin.plist"
rm "$HOME/Library/LaunchAgents/io.github.blue-flag-666.shtech-netlogin.plist"
```

## 常见问题

检查当前网络状态：

```sh
shtu-net-login status
```

- `online`：网络已连通，无需登录。
- `captive`：检测到校园网认证门户，可以执行 `login`。
- `offline`：没有可用网络，或未连接到校园网。

自动启动不工作时，先停止系统启动项，再在终端前台运行 `shtu-net-login watch` 查看具体错误。密码错误或账号被锁定时程序会停止，不会持续尝试。

只有在确认校园网门户证书确实异常时才临时使用 `--insecure`；该选项会关闭 TLS 证书验证。

## 从源码构建

需要 Go 1.27.1 或更新版本：

```sh
go test ./...
go build ./cmd/shtu-net-login
./scripts/build-all.sh v0.1.2
```

Windows PowerShell：

```powershell
go test ./...
.\scripts\build-all.ps1 -Version v0.1.2
```

第三方许可见 `THIRD_PARTY_NOTICES.md`。
