# 安装指南

ProfileWeave 是本地优先的浏览器 Profile 管理器。发布包只包含 ProfileWeave
服务端和已构建的管理台，不包含 Chromium、Firefox、Camoufox 或其他浏览器。
首次使用前请安装一个受支持的本地 Chromium 系浏览器，或在界面中选择自定义
浏览器可执行文件。

## 从 GitHub Release 安装

1. 在仓库 Releases 页面选择稳定版本。
2. 按操作系统和架构下载归档：
   - Windows：`profileweave-vX.Y.Z-windows-amd64.zip` 或 `arm64.zip`
   - Linux：`profileweave-vX.Y.Z-linux-amd64.tar.gz` 或 `arm64.tar.gz`
   - macOS：`profileweave-vX.Y.Z-darwin-amd64.tar.gz` 或 `arm64.tar.gz`
3. 同时下载 `SHA256SUMS`，在运行任何文件前校验归档。
4. 解压到仅当前用户可写的固定目录，不要从临时下载目录长期运行。

Windows 校验示例：

```powershell
Get-FileHash .\profileweave-vX.Y.Z-windows-amd64.zip -Algorithm SHA256
Select-String -Path .\SHA256SUMS -Pattern "profileweave-vX.Y.Z-windows-amd64.zip"
```

Linux 或 macOS 校验示例：

```bash
shasum -a 256 profileweave-vX.Y.Z-linux-amd64.tar.gz
grep 'profileweave-vX.Y.Z-linux-amd64.tar.gz' SHA256SUMS
```

两个哈希值必须完全一致。安装了 GitHub CLI 时，还可以验证 GitHub Actions
生成的构建 provenance：

```bash
gh attestation verify profileweave-vX.Y.Z-linux-amd64.tar.gz --repo <owner>/<repository>
```

归档目录应保持以下相对布局，服务端需要旁边的前端资源：

```text
profileweave-vX.Y.Z-<os>-<arch>/
├── profileweave[.exe]
├── frontend/
│   └── dist/
├── README.md
├── LICENSE
├── NOTICE
└── THIRD_PARTY_NOTICES.md
```

在该目录中运行 `profileweave.exe`（Windows）或 `./profileweave`
（Linux/macOS），再打开控制台输出的 loopback 地址。不要把本地 HTTP 服务通过
反向代理、端口转发或公网隧道暴露到局域网或互联网。

> 当前自动发布流程提供校验和、SBOM 和 GitHub provenance，但未配置 Windows
> Authenticode 或 Apple Developer ID 签名/公证。系统可能显示未知发布者提示。
> 不要绕过组织安全策略；需要平台签名时应等待签名发行版或从已审计源码构建。

## 从源码构建

要求：Go 1.25+、Node.js 22、pnpm 11 和 PowerShell 7（非 Windows 平台也需
`pwsh` 才能运行仓库脚本）。

```powershell
pnpm --dir frontend install --frozen-lockfile
pwsh -NoProfile -File scripts/verify.ps1
pwsh -NoProfile -File scripts/build.ps1 -SkipVerify
```

也可以分别构建：

```powershell
pnpm --dir frontend build
go build -trimpath -o dist/profileweave.exe ./cmd/server
```

直接运行源码版本时，默认从 `frontend/dist` 读取管理台资源。

## 数据与浏览器运行时

每个 Profile 使用独立的浏览器 user-data 目录。建议在投入使用前通过
`PROFILEWEAVE_DATA_DIR` 固定数据位置并纳入备份；从早期构建升级时也可能需要
保留兼容环境变量 `FINGERPRINT_BROWSER_DATA_DIR`。以当前版本 README 的配置表
为准。

ProfileWeave 只支持无认证 HTTP/SOCKS5 代理，不应在配置或问题报告中填写代理
用户名、密码或令牌。浏览器自身的许可、自动更新和安全补丁由用户与浏览器供应商
负责。
