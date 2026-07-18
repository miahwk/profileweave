# 安装指南

ProfileWeave 是本地优先的浏览器 Profile 管理器。发布包只包含 ProfileWeave
服务端和已构建的管理台，不包含 Chromium、Firefox、Camoufox 或其他浏览器。
首次使用前请安装一个受支持的本地 Chromium 系浏览器，或在界面中选择自定义
浏览器可执行文件。

## 从 GitHub Release 安装

> 当前尚无稳定 Release；首个 `v0.1.0` Windows 安装包按 unsigned preview 发布。
> 在此之前请按“从源码构建”操作，不要把分支快照当作正式发行包。

1. 在仓库 Releases 页面选择目标版本。当前未签名版本应显示为 Pre-release，不能当作
   已签名稳定版本；先阅读 Release Notes 中的安全边界。
2. 按操作系统和架构下载产物：
   - Windows 推荐：`profileweave-vX.Y.Z-windows-amd64-setup.exe` 或
     `profileweave-vX.Y.Z-windows-arm64-setup.exe`
   - Windows 便携包：`profileweave-vX.Y.Z-windows-amd64.zip` 或 `arm64.zip`
   - Linux：`profileweave-vX.Y.Z-linux-amd64.tar.gz` 或 `arm64.tar.gz`
   - macOS：`profileweave-vX.Y.Z-darwin-amd64.tar.gz` 或 `arm64.tar.gz`
3. 同时下载 `SHA256SUMS`，在运行任何文件前校验目标产物。
4. Windows setup 校验后直接运行安装；ZIP/TAR 便携归档则解压到仅当前用户可写的
   固定目录，不要从临时下载目录长期运行。

Windows 安装器校验示例：

```powershell
$artifact = '.\profileweave-vX.Y.Z-windows-amd64-setup.exe'
Get-FileHash -LiteralPath $artifact -Algorithm SHA256
Select-String -Path .\SHA256SUMS -SimpleMatch ([IO.Path]::GetFileName($artifact))
gh attestation verify $artifact --repo miahwk/profileweave
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

### Windows 安装器

系统要求为 Windows 10 version 1809（build 17763）或 Windows Server 2019 及更高
版本；amd64 与 arm64 必须下载各自匹配的安装包。

安装器只为当前用户安装到 `%LOCALAPPDATA%\Programs\ProfileWeave`，不需要管理员权限。
默认创建开始菜单中的“ProfileWeave”和“Exit ProfileWeave”，桌面快捷方式为安装时的
可选项。启动快捷方式会在本地服务监听成功后打开默认浏览器；重复点击会先验证端口上
的 ProfileWeave 身份，再复用已有实例。

卸载程序会先请求应用正常退出，再删除程序文件和快捷方式。它**不会删除**
`%APPDATA%\ProfileWeave`；升级和卸载后，Profile 元数据及浏览器 user-data 都会保留。
完整清理必须在备份后由用户手动执行，不能让卸载器递归删除未核实的用户数据目录。

当前 setup 和 payload 未配置 Authenticode。Windows 可能显示未知发布者或 SmartScreen
提示；这不是校验失败的替代判断。应核对 SHA-256 与 GitHub provenance，并遵守组织
安全策略。无法接受未签名程序的环境请从已审计源码构建或等待签名发行版。

### 便携归档

归档目录应保持以下相对布局，服务端需要旁边的前端资源：

```text
profileweave-vX.Y.Z-<os>-<arch>/
├── profileweave[.exe]
├── frontend/
│   └── dist/
├── README.md
├── LICENSE
├── NOTICE
├── THIRD_PARTY_NOTICES.md
├── CHANGELOG.md
└── SECURITY.md
```

在该目录中运行 `profileweave.exe --open`（Windows）或 `./profileweave --open`
（Linux/macOS）。也可不加 `--open`，再手动打开控制台输出的 loopback 地址。不要把本地 HTTP 服务通过
反向代理、端口转发或公网隧道暴露到局域网或互联网。

> 当前自动发布流程提供校验和、SBOM 和 GitHub provenance，但未配置 Windows
> Authenticode 或 Apple Developer ID 签名/公证。系统可能显示未知发布者提示。
> 不要绕过组织安全策略；需要平台签名时应等待签名发行版或从已审计源码构建。

## 从源码构建

要求：Go 1.25.12+、Node.js 22、pnpm 11 和 PowerShell 7（非 Windows 平台也需
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
