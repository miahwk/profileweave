# ProfileWeave Browser

ProfileWeave Browser 是一个使用 Vue 3 与 Go 构建的本地浏览器 Profile 隔离和运行控制面。它面向授权 QA、隐私研究与日常会话隔离：为每个 Profile 保存独立浏览器数据目录，检查语言、时区、屏幕、代理与 UA 等配置是否自洽，并负责本机浏览器的启动和停止。

> 本项目交付的是存储隔离、配置诊断和本地进程管理，不承诺绕过网站风控。当前版本不伪造 Canvas、WebGL、Audio、字体、GPU 或 TLS，也不提供 CAPTCHA、批量账号或站点专用规避逻辑。

## 已实现能力

- Profile 创建、编辑、搜索、复制、可恢复删除、回收站恢复/永久删除和 revision 冲突保护。
- Chrome、Edge、Brave、Chromium 自动发现及自定义浏览器路径。
- 每个 Profile 独立 `user-data-dir`，隔离 cookies、localStorage、cache 和站点权限。
- direct、HTTP、SOCKS5 无认证代理，以及限制非代理 WebRTC UDP 的隐私策略。
- OS/UA、locale/languages、IANA timezone、screen/DPR、CPU/内存意图、proxy/WebRTC 一致性评分。
- “已应用 / 部分应用 / 未支持”的运行时能力矩阵，不把已保存配置误报为内核能力。
- 本地 JSON 原子持久化、loopback API、随机写操作令牌、Host/Origin 防护、安全 argv 启动。
- Vue 响应式管理台、Go/TypeScript 测试、CI 和源文件行数守卫。

设计与验收资料：

- [高 Star 开源市场对比](docs/market-comparison-2026-07.md)
- [开源技术路线调研](docs/open-source-research.md)
- [DDD 实现方案](docs/architecture.md)
- [交付价值与验收](docs/delivery-value.md)
- [威胁模型](docs/threat-model.md)

## 快速开始

开发环境要求 Go 1.25.12+、Node.js 22、pnpm 11，以及至少一个本机 Chromium 系浏览器。

```powershell
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend build
go run ./cmd/server
```

打开 `http://127.0.0.1:3210`。开发前端时，可在另一个终端运行 `pnpm --dir frontend dev`，然后访问 `http://127.0.0.1:5173`。

## 验证与构建

```powershell
powershell -ExecutionPolicy Bypass -File scripts/verify.ps1
powershell -ExecutionPolicy Bypass -File scripts/build.ps1 -SkipVerify -Version 0.1.0
dist/profileweave.exe --version
```

本地构建输出到 `dist/`，包含可执行文件、Vue 静态资源、README、项目许可证和第三方许可说明。正式 Release 由 `v*` tag 工作流生成多平台归档、SHA256 校验和、SBOM 与构建来源证明；详见[发布流程](docs/release-process.md)。

## 配置

新名称环境变量优先；旧的 `FINGERPRINT_BROWSER_*` 名称仅为迁移兼容而保留。

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PROFILEWEAVE_PORT` | `3210` | loopback HTTP 端口 |
| `PROFILEWEAVE_DATA_DIR` | OS 用户配置目录下的 `ProfileWeave` | `profiles.json`、浏览器数据与回收区 |
| `PROFILEWEAVE_WEB_DIR` | 可执行文件相邻的 `frontend/dist`，开发时为仓库的 `frontend/dist` | Vue production build 目录 |

数据结构：

```text
ProfileWeave/
├── profiles.json
├── browser-data/
│   ├── p_<profile-id>/
│   └── ...
└── trash/
    └── browser-data/
        └── <opaque-restore-token>/
```

代理凭据不会持久化，因此当前版本只接受无认证 HTTP/SOCKS5 代理。

## 平台状态

| 平台 | 构建/CI | 浏览器实机验证 | 发布说明 |
| --- | --- | --- | --- |
| Windows amd64 | 支持 | Chrome、Edge | 首要支持平台；正式签名前二进制会显示 unsigned 提示 |
| Windows arm64 | 交叉构建 | 未验证 | 预览支持 |
| Linux amd64/arm64 | CI 与交叉构建 | 未完成完整矩阵 | 预览支持 |
| macOS amd64/arm64 | CI 与交叉构建 | 未完成完整矩阵 | 预览支持；尚未 notarize |

## 已知限制

- OS target、languages、timezone、CPU 和内存当前用于保存与诊断，尚未注入浏览器。
- `--lang`、窗口尺寸、DPR、代理和 WebRTC flags 的最终行为取决于浏览器版本与操作系统。
- 自定义 UA 不能同步全部 User-Agent Client Hints 或 TLS 特征，系统会显示警告。
- 代理不是 VPN，不保证覆盖全部 DNS、扩展或系统层流量。
- Session 是内存状态；服务重启后不会根据历史 PID 接管未知进程。
- 第三方运行时必须逐项审查源码、许可证和再分发权；CloakBrowser 只能由用户自带，不能随本项目分发。

## API 摘要

API 前缀为 `/api/v1`：

- `GET /health`、`GET /capabilities`
- `GET /bootstrap`（取得本进程临时写操作令牌）
- `GET|POST /profiles`
- `GET|PUT|DELETE /profiles/{id}`
- `POST /profiles/{id}/duplicate|validate|launch|stop`
- `GET /sessions`

服务固定监听 loopback。不要通过反向代理把本地控制 API 暴露到局域网或公网。完整错误结构和兼容性约定见 [OpenAPI 契约](api/openapi.yaml)。

所有非安全方法还必须携带 `X-ProfileWeave-Token`。管理台会从同源 `/bootstrap` 自动获取；CLI 调用者应在每次服务重启后重新获取，不得记录或持久化该令牌。

## 安全、贡献与许可

提交安全问题前请阅读 [SECURITY.md](SECURITY.md)。开发流程见 [CONTRIBUTING.md](CONTRIBUTING.md)，社区行为约定见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

项目以 [Apache License 2.0](LICENSE) 发布；分发时必须同时保留 [NOTICE](NOTICE) 与 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
