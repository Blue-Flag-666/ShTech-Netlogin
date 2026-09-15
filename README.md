# shtu-net-login

上海科技大学校园网自动认证工具。根据浏览器 HAR 中的新门户流程实现，验证码在本机通过内嵌模型识别；无需浏览器、Python、ONNX Runtime 或其他动态库。

> 本项目是非官方工具。请只使用自己的校园网账号，并遵守学校网络使用规定。

## 功能

- Windows、Linux、macOS；amd64、arm64，额外支持 Linux armv7
- 单个可执行文件，无运行时依赖
- `login` 单次登录、`watch` 断线监测并自动重登、`status` 状态检测
- 自动从门户重定向提取 `pushPageId`、SSID、客户端 IP 和 AC 地址
- 内嵌纯 Go 验证码推理，不调用外部 OCR 服务
- 密码错误或账号锁定时停止重试，避免扩大锁定风险
- 兼容参考项目的 `EGATE_ID`、`EGATE_PASSWORD` 环境变量

## 快速使用

推荐通过环境变量提供凭据，密码不会出现在命令行参数或进程列表中。

PowerShell：

```powershell
$env:SHTU_USERNAME = "你的学号"
$env:SHTU_PASSWORD = "你的密码"
.\shtu-net-login-windows-amd64.exe login
.\shtu-net-login-windows-amd64.exe watch
```

Linux/macOS：

```sh
export SHTU_USERNAME='你的学号'
export SHTU_PASSWORD='你的密码'
./shtu-net-login-linux-amd64 login
./shtu-net-login-linux-amd64 watch
```

也可以把 `config.example.json` 复制到系统配置目录：

- Windows：`%AppData%\shtu-net-login\config.json`
- Linux：`$XDG_CONFIG_HOME/shtu-net-login/config.json`，未设置时为 `~/.config/shtu-net-login/config.json`
- macOS：`~/Library/Application Support/shtu-net-login/config.json`

Linux/macOS 上请将包含密码的配置文件权限设为仅本人可读：

```sh
chmod 600 ~/.config/shtu-net-login/config.json
```

## 命令

```text
shtu-net-login login          检测网络，需要认证时登录（默认命令）
shtu-net-login watch          持续监测，掉线后自动登录
shtu-net-login status         输出 online、captive 或 offline
shtu-net-login sample-config  输出示例配置
shtu-net-login version        输出版本
```

通用选项需放在命令之后：

```text
--config PATH       指定配置文件
--interval 30s      watch 检查间隔
--timeout 12s       HTTP 请求超时
--base-url URL      覆盖门户地址，主要用于测试
--insecure          跳过 TLS 证书校验，仅在证书异常时临时使用
```

环境变量优先于配置文件：`SHTU_USERNAME`、`SHTU_PASSWORD`、`SHTU_BASE_URL`、`SHTU_INTERVAL`、`SHTU_TIMEOUT`、`SHTU_INSECURE`。也兼容 `EGATE_ID` 和 `EGATE_PASSWORD`。

## 开机自动运行

先确认 `watch` 在终端中工作，再按系统建立启动项：

- Windows：使用“任务计划程序”，触发器选“用户登录时”，操作指向对应 exe，参数填 `watch`；凭据建议设置为该用户的环境变量或保存到配置文件。
- Linux：建立用户级 systemd 服务，`ExecStart` 指向二进制并附加 `watch`，然后执行 `systemctl --user enable --now ...`。
- macOS：建立用户 LaunchAgent，`ProgramArguments` 中写入二进制路径和 `watch`。

不要在多人可读的服务定义中直接写密码。

## 构建与测试

需要 Go 1.26.1 或更新版本：

```sh
go test ./...
go build ./cmd/shtu-net-login
./scripts/build-all.sh v0.1.0
```

Windows PowerShell：

```powershell
go test ./...
.\scripts\build-all.ps1 -Version v0.1.0
```

构建脚本会在 `dist/` 生成 Windows/Linux/macOS 的 amd64、arm64 文件，以及 Linux armv7 文件。

## GitHub Actions 自动发布

仓库内的 `.github/workflows/release.yml` 会完成测试、七目标交叉编译、SHA-256 校验及 Release 上传。发布版本时推送一个符合语义化版本格式的标签：

```sh
git tag v0.1.0
git push origin v0.1.0
```

工作流结束后，七个可执行文件及各自的 `.sha256` 校验文件会出现在该标签对应的 GitHub Release 中。也可以从 Actions 页面手动运行工作流，但输入的标签必须已经存在于远端仓库。

工作流只在最终发布任务中授予 `contents: write`，测试和构建任务保持只读权限；发布使用 GitHub runner 自带的 `gh` 和仓库范围的 `GITHUB_TOKEN`，不需要额外配置个人令牌。

## 实现依据

认证请求字段和响应判断同时参考了用户提供的 HAR 与 [`ShanghaitechGeekPie/net-loginer`](https://github.com/ShanghaitechGeekPie/net-loginer)。本实现没有复制 HAR 中的账号、密码、Cookie、令牌或会话标识。

第三方许可见 `THIRD_PARTY_NOTICES.md`。
