# Windows 安装与应用生命周期 RFC

状态：已接受，目标版本 `v0.1.0` preview，2026-07-18。

## 1. 目标与边界

本阶段让 Windows 用户能够使用普通用户权限安装、启动、升级和卸载 ProfileWeave，
并在整个生命周期保留 `%APPDATA%\ProfileWeave` 下的 Profile 元数据和浏览器数据。
安装包仍然不捆绑 Chromium、Edge 或其他浏览器，也不改变“本地 Profile 隔离和配置
一致性”这一产品边界。

首个安装包是**未签名 preview**。发布流程提供 SHA-256、SBOM 和 GitHub build
provenance，但在配置 Authenticode 证书之前，Windows 可能显示“未知发布者”或
SmartScreen 提示。文档和 Release Notes 必须如实说明，不能把未签名安装包描述为
低摩擦的稳定发行版。

## 2. 安装器选型

| 方案 | 优点 | 首发缺点 | 决策 |
| --- | --- | --- | --- |
| Inno Setup 6.7.3 | 原生支持 per-user、x64/Arm64、快捷方式、同 AppId 升级、静默安装/卸载 | 官网建议商业用户购买许可；仍需自行配置 Authenticode | **采用并固定 6.7.3** |
| NSIS 3.12 | zlib/libpng 风格许可，脚本自由度高 | Arm64 仍较早期；升级、卸载注册和架构规则需要更多底层脚本 | 备选 |
| WiX 7 | MSI 修复、事务和企业部署能力强 | 组件/升级模型复杂；当前二进制发行受 OSMF 条款约束 | 有明确企业 MSI 需求时再评估 |
| MSIX | 干净卸载、差分更新、Store/AppInstaller 分发 | 生产安装必须签名并建立证书信任；还需验证容器对本地 HTTP 和外部浏览器启动的影响 | 建立签名/Store 渠道后再评估 |

主要上游依据：

- [Inno Setup 下载与稳定版本](https://jrsoftware.org/isdl.php)
- [Inno Setup 许可](https://raw.githubusercontent.com/jrsoftware/issrc/is-6_7_3/license.txt)
- [Inno Setup 架构约束](https://jrsoftware.org/ishelp/topic_setup_architecturesallowed.htm)
- [Inno Setup AppId 与升级识别](https://jrsoftware.org/ishelp/topic_setup_appid.htm)
- [NSIS 许可](https://nsis.sourceforge.io/Docs/AppendixI.html)
- [WiX OSMF](https://docs.firegiant.com/wix/osmf/)
- [MSIX 签名要求](https://learn.microsoft.com/windows/msix/package/signing-package-overview)

## 3. 安装布局

安装器按架构分别产出：

```text
profileweave-vX.Y.Z-windows-amd64-setup.exe
profileweave-vX.Y.Z-windows-arm64-setup.exe
```

- 安装范围：当前用户，不申请管理员权限。
- 最低系统版本：Windows 10 version 1809（build 17763）或 Windows Server 2019。
- 固定安装目录：`%LOCALAPPDATA%\Programs\ProfileWeave`。
- 固定产品 AppId：升级时保持不变，覆盖应用文件并复用卸载记录。
- 默认创建开始菜单快捷方式；桌面快捷方式由用户显式选择。
- 安装文件包含 GUI 子系统的 `profileweave.exe`、Vue 静态资源、README、许可证、
  notices、changelog 和安全说明。
- 用户数据继续由 Go 的 `os.UserConfigDir()` 解析到 `%APPDATA%\ProfileWeave`。
  本轮不无迁移地改变已有数据路径。

应用文件和用户数据必须分离。安装器只拥有安装目录，不把
`%APPDATA%\ProfileWeave` 写入删除清单；升级和卸载均保留 Profile、回收站和浏览器
user-data。用户如需彻底清理，应先备份，再按文档手动删除已经核实的目录。

## 4. 启动、重复启动与退出

开始菜单快捷方式执行：

```text
profileweave.exe --open --log-file %LOCALAPPDATA%\ProfileWeave\logs\profileweave.log
```

应用先同步绑定 `127.0.0.1:<port>`，成功监听后才通过 Windows ShellExecute 打开固定
loopback 管理台 URL，不调用 `cmd /c start`，也不把用户输入拼接到 shell 命令中。

如果同一数据目录已经被 ProfileWeave 持有，第二次点击快捷方式不得只显示锁错误。
它必须先请求 `/api/v1/health` 并验证响应的产品标识，确认端口上确实是 ProfileWeave
后再打开既有管理台。端口被其他程序占用时，不打开、不终止该程序。

安装版使用 Windows GUI 子系统，避免常驻控制台窗口。运行日志写入有限大小的本地
日志文件。管理台提供带二次确认的“退出应用”；受现有 loopback Host/Origin 和随机
控制令牌保护的 `POST /api/v1/shutdown` 触发统一清理。开始菜单同时提供退出快捷方式，
其 `--shutdown` 命令只在验证健康端点产品标识并取得本地控制令牌后请求退出。

正常信号、管理台退出、退出命令和 HTTP Serve 错误进入同一清理路径：先关闭新的
Launch 入口并等待当前生命周期临界区，再停止由本应用持有的浏览器会话；清理成功后
才停止 HTTP 并释放数据目录锁。浏览器停止失败时保持服务和锁以供重试，使安装/卸载
命令超时失败，而不是留下不受管理的会话后误报退出成功。不得在服务 goroutine 中调用
`log.Fatal` 绕过清理。

## 5. 升级与卸载

- 同一 AppId、安装目录和架构的新版执行覆盖升级，用户数据不移动。
- 跨架构安装使用同一 AppId，但必须由目标架构安装器检查当前 Windows 架构。
- 卸载前先运行 `profileweave.exe --shutdown` 并等待统一清理完成。
- 卸载删除应用文件、开始菜单/桌面快捷方式和卸载注册信息，但不删除用户数据。
- 退出失败时不根据未验证 PID 强杀进程；卸载器应报告占用，让用户关闭应用后重试。

## 6. 发布与供应链

Release 工作流在原生 Windows runner 上：

1. 下载 quality job 已验证的前端 bundle。
2. 从 Inno Setup 上游固定 release 下载 6.7.3，并验证固定 SHA-256 后静默安装编译器。
3. 分别构建 `windowsgui` amd64/arm64 payload 和安装器。
4. 执行静默安装、版本检查、启动/health、退出、卸载和数据保留 smoke test。
5. 将 setup EXE 与现有 ZIP 一并纳入 SHA256SUMS、SBOM、provenance 和 GitHub Release。

源码仓库不存放签名证书、私钥或长期令牌。将来配置 Authenticode 时，必须在受保护的
发布环境使用短期凭据，并在生成最终校验和与 provenance 之前签署 payload、setup 和
uninstaller。

## 7. 验收标准

- amd64 安装器可在非管理员上下文静默安装，安装后的 `--version` 与 tag 一致。
- 安装后启动会打开 loopback 管理台；第二次启动复用既有实例。
- 健康端点身份不匹配时，`--open` 和 `--shutdown` 均安全失败。
- 管理台退出与 `--shutdown` 都会停止 HTTP 服务和受管浏览器会话。
- 同 AppId 覆盖升级后，预先创建的数据 sentinel/Profile 仍存在。
- 静默卸载后安装目录和快捷方式消失，隔离测试数据目录中的 sentinel 仍存在。
- arm64 安装器至少完成交叉构建；在可用原生 Arm runner 后执行同一套 smoke。
- `go test ./...`、`go vet ./...`、前端测试/构建和行数检查全部通过。

## 8. 暂不包含

- Authenticode 证书购买、SmartScreen reputation 和正式 stable 签名承诺。
- 自动更新器、后台 Windows Service、系统托盘和开机自启动。
- 浏览器二进制下载、捆绑或自动更新。
- 安装器内“一键删除全部 Profile 数据”。
