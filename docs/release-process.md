# 发布流程

本文面向仓库维护者，定义可重复、可审计的 GitHub Release 过程。Release 工作流
构建 Windows、Linux、macOS 的 amd64/arm64 归档和 Windows per-user 安装器，发布
SHA-256、SPDX JSON SBOM 并生成 GitHub build provenance。

> 当前尚未发布稳定版本。`v0.1.0` 仍处于 `Unreleased`，首个未签名安装包必须发布为
> Pre-release；只有完成本文全部发布前检查、从已验证的默认分支提交创建 tag，并确认
> Release 工作流成功后，才能标记为已发布的 preview。

## 仓库一次性配置

公开发布前，仓库所有者应完成：

- 启用 GitHub Private Vulnerability Reporting。
- 对默认分支启用保护规则，要求 `CI` 的多平台验证和依赖安全检查通过。
- 仅允许受信任维护者创建 `v*` tag 和 GitHub Release。
- 保持 GitHub Actions 默认 `GITHUB_TOKEN` 最小权限；只有发布 job 获得
  `contents: write`、`id-token: write` 与 `attestations: write`。
- 配置 Dependabot，并定期处理 Go、npm 与 GitHub Actions 更新。
- 在仓库确定最终 owner/name 后检查 README、issue 表单和 changelog 中的链接。

当前工作流没有私钥，也没有平台代码签名。若配置 Windows Authenticode 或 Apple
Developer ID，使用受保护的环境与短期凭证，在归档和生成校验和之前签名；不要把
证书或长期密钥提交到仓库。

## 构建信息契约

发布构建通过 Go linker 注入以下变量：

```text
github.com/miahwk/profileweave/internal/buildinfo.Version
github.com/miahwk/profileweave/internal/buildinfo.Commit
github.com/miahwk/profileweave/internal/buildinfo.Date
```

这三个变量必须是包级可写字符串，并提供非 Release 构建使用的安全默认值（例如
`dev`、`unknown`）。在创建首个 tag 前必须确认该包存在且健康检查或版本命令能显示
这些值。模块路径变更时要同步更新 `.github/workflows/release.yml` 的 `-X` 路径。

## 发布前检查

1. 从默认分支创建聚焦的发布 PR。
2. 更新 `CHANGELOG.md`：把 Unreleased 项移动到新版本并填写发布日期。
3. 确认 README、安装、升级、安全边界和第三方 notices 与实际行为一致。
4. 若依赖或捆绑内容改变，更新 `THIRD_PARTY_NOTICES.md` 并复核许可证兼容性。
5. 确认持久化格式向后兼容，或已经提供、测试并记录显式迁移。
6. 本地执行：

```powershell
pwsh -NoProfile -File scripts/verify.ps1
pnpm --dir frontend audit --audit-level high
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

应用与 `govulncheck` v1.6.0 统一使用 Go 1.25.12 或更高的安全补丁工具链；`go.mod`
是 CI 和 Release 的唯一工具链版本源。

7. 合并后确认默认分支 CI 全绿，工作区无未提交变更。
8. 在隔离的临时数据目录执行一次打包产物的 Profile 创建、启动、停止和重启恢复
   smoke test。不得使用真实账号或生产 Profile。
9. 在原生 Windows runner 对 setup 执行静默安装、`--version`、启动/health、
   同 AppId 覆盖安装、`--shutdown` 和静默卸载；确认安装目录删除而隔离测试数据目录中
   的 sentinel 保留。

## 创建版本

版本使用 Semantic Versioning。由默认分支的已验证 commit 创建签名或 annotated tag：

```bash
git tag -s vX.Y.Z -m "ProfileWeave vX.Y.Z"
git push origin vX.Y.Z
```

没有可用签名身份时可创建 annotated tag，但应在 Release Notes 中明确 tag 未签名。
推送匹配 `v*` 的 tag 会触发 Release 工作流。工作流会拒绝非
`vMAJOR.MINOR.PATCH`（允许 SemVer 后缀）格式。

## 验证自动发布

发布 job 完成后，维护者必须：

- 确认六个 OS/架构归档、两个 Windows setup、`SHA256SUMS` 和一个 SPDX JSON SBOM 都已上传。
- 在干净环境下载归档并独立计算 SHA-256，与 `SHA256SUMS` 比较。
- 使用 `gh attestation verify <artifact> --repo <owner>/<repository>` 验证 provenance。
- 解压至少 Windows、Linux、macOS 各一个原生架构包，检查 LICENSE、NOTICE、
  THIRD_PARTY_NOTICES、README 和 `frontend/dist` 都存在。
- 在支持的平台运行二进制并确认版本、commit、build date 与 tag/commit 一致。
- 核对 Windows setup 文件名架构、普通用户安装目录、开始菜单启动/退出快捷方式和
  卸载数据保留行为；未签名时把 Release 标为 preview 并明确 SmartScreen 风险。
- 确认发布包没有浏览器二进制、测试 Profile、缓存、日志、`.env` 或凭证。

若任何验证失败，删除或标记该 Release 为预发布并发布修复版本；不要静默替换同名
已发布归档，因为这会使已有校验和与 provenance 失效。

## 发布后

- 更新安装渠道或文档中的稳定版本链接。
- 观察公开 issue 与 Private Vulnerability Reporting，但不要要求用户公开敏感日志。
- 将下一版本的 `CHANGELOG.md` 保持为 Unreleased。
- 保存 GitHub Actions 运行记录、Release 资产、SBOM 与 provenance；发布产物不应只
  存在于短期 CI artifact 中。
