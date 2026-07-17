# 高 Star 开源浏览器方案对比（2026-07）

> 快照日期：2026-07-17。GitHub 页面显示的 star 为取整后的动态快照，只反映社区关注度，不能直接等同于安全性、开源完整度或产品成熟度。

## 结论

市场上没有一个项目同时完美覆盖“完整开源内核、成熟桌面管理器、Go 控制面、稳定生产发布”。ProfileWeave 因此采用组合路线：

- 产品与 Profile 生命周期参考 Donut Browser；
- Go 进程控制、本地 API、安全边界与 doctor 思路参考 PinchTab；
- fully-open 原生运行时优先评估 Camoufox；
- 一致性规则参考 Apify fingerprint-suite 与 BrowserForge；
- 默认隐私策略参考 Mullvad Browser 与 arkenfox；
- CloakBrowser 只允许用户自带，不下载、不镜像、不随包分发。

## Star、许可与开放程度

| 项目 | Star 快照 | 类型 | 许可证/开放程度 | 对 ProfileWeave 的价值 |
| --- | ---: | --- | --- | --- |
| [CloakBrowser](https://github.com/CloakHQ/CloakBrowser) | 28.5k | Chromium stealth wrapper | wrapper MIT；核心二进制专有且限制再分发 | 借 provider 接口与持久 seed；只能 BYO |
| [FingerprintJS](https://github.com/fingerprintjs/fingerprintjs) | 27.9k | 指纹检测库，不是浏览器 | MIT 客户端库；商业识别平台另计 | 可作本地离线观测工具，不能当运行时 |
| [arkenfox/user.js](https://github.com/arkenfox/user.js) | 12.7k | Firefox 隐私加固配置 | MIT，完全开放但不是管理器 | 借 baseline/override/迁移方法 |
| [Betterfox](https://github.com/yokoffing/Betterfox) | 10.6k | Firefox 日用配置 | MIT，配置层 | 借兼容性友好的隐私默认值 |
| [Camoufox](https://github.com/daijro/camoufox) | 10.2k | Firefox 原生补丁内核 | MPL-2.0，补丁与构建工具开放 | 首个 fully-open runtime 候选 |
| [PinchTab](https://github.com/pinchtab/pinchtab) | 9.4k | Go 浏览器控制面 | MIT；默认依赖外部 Chrome | 最直接的 Go 工程与安全对标 |
| [Donut Browser](https://github.com/zhom/donutbrowser) | 3.4k | 桌面 Profile 管理器 | AGPL-3.0；Wayfern 内核开放性需另审 | 产品功能与 UX 对标 |
| [Apify fingerprint-suite](https://github.com/apify/fingerprint-suite) | 2.5k | 指纹生成/注入工具包 | Apache-2.0 | 借关联约束，不把 JSON 误报为已应用 |
| [Mullvad Browser](https://github.com/mullvad/mullvad-browser) | 2.4k | 反指纹隐私浏览器 | MPL-2.0，完整浏览器源码开放 | 借“减少可配置面、让用户趋同”原则 |
| [BrowserForge](https://github.com/daijro/browserforge) | 1.2k | Python 指纹/头生成器 | Apache-2.0；注入功能已 deprecated | 借现实分布生成；不采用页面注入承诺 |

另有 [Kameleo](https://github.com/kameleo-io/kameleo) 等 SDK/文档仓，但核心运行时闭源，不作为开源实现基线。

## 代表项目分析

### Camoufox：fully-open 内核首选候选

优点是公开 Firefox patch、构建工具、包装器和 MPL-2.0 许可，可自行审计和构建；缺点是仍处快速开发期，跟随浏览器安全更新与多平台兼容的成本高。适合作为可选 provider，不宜在未经版本固定和兼容矩阵验证前替换系统 Chromium 默认模式。

### PinchTab：Go 控制面的工程基线

它的单二进制、实例/profile/tab 分层、HTTP/CLI/MCP、doctor、loopback 默认值和敏感 API 提示，最适合本仓后端对标。它并不解决原生指纹内核问题，Windows 也不是其最强平台，因此应借工程方法，不照搬产品承诺。

### Donut Browser：功能价值标杆

Donut 已覆盖 groups、批量设置、多协议代理、cookie/扩展/profile 导入导出、REST/MCP 与自托管同步，是最接近本项目形态的开源管理器。其管理器为 AGPL-3.0，但公开资料不足以证明 Wayfern C++ 内核完全开放；本项目只参考用户工作流，不复制代码或捆绑未审清运行时。

### CloakBrowser：高 Star 不等于完全开源

公开仓库主要是多语言 wrapper。核心 Chromium 修改没有随仓开放，二进制有独立许可且限制再分发。正确集成方式只能是用户自行提供二进制，并在 UI 明确显示第三方来源、版本、能力与许可。

### Apify/BrowserForge：生成与应用必须分开

两者擅长生成相互关联的 UA、headers、OS、设备、屏幕和硬件画像，但生成出的数据不等于浏览器已经具备对应信号。ProfileWeave 只借鉴约束模型：运行时真正支持的字段标记 `applied`，诊断字段标记 `diagnostic-only`，其余显示 `unsupported` 或 `conflicting`。

## 核心能力矩阵

| 能力 | ProfileWeave 0.1 | Donut | PinchTab | Camoufox | Cloak | Apify/BrowserForge |
| --- | --- | --- | --- | --- | --- | --- |
| GUI / Profile CRUD | 已实现 | 完整 | 基础 | 无 | 无完整管理器 | 无 |
| 持久 Profile 隔离 | 已实现 | 是 | 是 | 可由 context 实现 | 是 | 调用方负责 |
| Groups/批量操作 | 规划中 | 是 | 有限 | 否 | 否 | 否 |
| 原生内核处理 | 未捆绑 | Wayfern，需另审 | 默认无 | fully-open Firefox patch | 专有核心 | 无 |
| 一致性诊断 | 已实现规则集 | 有 | 非核心 | 有 | 有 | 强项 |
| REST/CLI/MCP | REST | REST/MCP | HTTP/CLI/MCP | Playwright | CDP/自动化 wrapper | 库 API |
| Cookie/扩展导入导出 | 规划中 | 是 | 基础持久化 | 调用方负责 | context 负责 | 调用方负责 |
| 同步/团队 | 未实现 | 自托管/E2E | 无 | 无 | 无 | 无 |
| 完整开放内核 | 不捆绑内核 | 未证实 | 不包含内核 | 是 | 否 | 不适用 |

## 差距与发布路线

### P0：开源 Release 基线

- 完整许可证、第三方 notices、SECURITY、威胁模型与贡献治理。
- profile 数据生命周期与可恢复删除，运行中禁止修改/删除。
- loopback Host/Origin 防护、安全响应头、API 禁缓存。
- build version/commit/date、`--version`、checksums、SBOM、provenance。
- 多平台 CI、依赖审计、`govulncheck`、归档冒烟验证。

### P1：达到同类产品核心价值

- runtime provider 模型及来源/版本/许可证/能力展示。
- groups、tags 与批量启动/停止/代理修改。
- profile、cookie、扩展导入导出。
- proxy connectivity test、启动日志、doctor 和 crash reconciliation。
- Camoufox 固定版本适配器与兼容矩阵；CloakBrowser 仅 BYO。

### P2：稳定后评估

- OS 密钥链凭据、可选 GeoIP provider、运行时下载校验与回滚。
- E2E 加密同步、团队审计、多机 bridge 和 MCP。

不以“通过某检测站”“不可检测”或 CAPTCHA 分数作为验收项。可复现验收只检查同 Profile 重启稳定、不同 Profile 存储隔离、配置与实际能力声明一致。
