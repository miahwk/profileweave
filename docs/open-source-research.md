# 开源指纹浏览器方案调研

> 调研日期：2026-07-17
> 目标：为合法的 QA、隐私研究和会话隔离工具选择可审计、可维护的技术路线；不以绕过目标网站风控为产品承诺。

## 1. 结论先行

开源生态大致分为四条路线：

1. **浏览器内核补丁**：在 Chromium/Firefox 的 C++ 层修改可观察信号。覆盖最完整，但维护和供应链成本最高。
2. **真实分布的指纹生成器**：生成 UA、请求头、屏幕、GPU 等相互有关联的数据。擅长“生成一致数据”，本身不负责可靠应用。
3. **Playwright/Puppeteer/CDP 注入**：开发快、跨平台，但注入痕迹、Worker/iframe/网络层不一致等问题无法彻底消除。
4. **隐私浏览器的同质化/随机化**：Mullvad/Tor 倾向让用户进入少量相同桶，Brave 倾向按站点和会话稳定随机化。它们更适合隐私保护，不适合任意伪装成另一台设备。

本项目取长补短后的选择是：**先交付本地优先的配置、隔离、生命周期和一致性诊断；浏览器内核能力做成可替换端口。MVP 使用用户已安装的 Chromium 系浏览器，不伪造 canvas/WebGL/audio，不宣称“不可检测”。** 未来若确有合法业务需要，再通过独立适配器集成经过许可证与供应链审查的内核。

## 2. 代表性方案对比

| 方案 | 技术路线与许可证 | 优点 | 缺点/风险 | 本项目借鉴 |
| --- | --- | --- | --- | --- |
| [Camoufox](https://github.com/daijro/camoufox) | Firefox C++/配置层补丁 + Python/Playwright；MPL-2.0 | 内核层修改，覆盖 WebGL、字体、WebRTC 等；使用 BrowserForge 生成现实分布；Playwright 接口成熟 | 官方仓库明确提示仍在开发；Firefox/SpiderMonkey 特征不能伪装成 Chromium；跟随 Firefox 升级成本高 | 内核能力必须适配器化；“现实分布 + 跨字段一致性”应成为领域规则；不要在 UI 层散落参数 |
| [Donut Browser](https://github.com/zhom/donutbrowser) | 桌面 Profile 管理器；AGPL-3.0；当前使用 Wayfern Chromium | 完整的 profile/proxy/group 管理、导入导出、同步和本地 API；产品形态最接近开源竞品 | AGPL 有强 copyleft 义务；公开仓库未证明 Wayfern C++ 内核完全开源，需单独审计 | 借鉴“管理器与内核解耦”、本地优先、profile/group 生命周期；不复制 AGPL 代码 |
| [CloakBrowser](https://github.com/CloakHQ/CloakBrowser) | Python/TypeScript/C# wrapper MIT；浏览器二进制为单独专有许可 | 持久 profile、固定 seed、CDP 与多自动化客户端接口 | 核心 Chromium C++ 补丁不开放；旧二进制禁止再分发，新版需 Pro 订阅 | 只借鉴 provider 接口；仅允许用户自带，不捕获、镜像或随包分发 |
| [BrowserForge](https://github.com/daijro/browserforge) | Python 指纹与请求头生成器；Apache-2.0 | 用贝叶斯生成网络模仿真实设备联合分布；约束浏览器、OS、设备、语言 | 只生成数据；应用到浏览器仍需其他技术；Python 依赖不符合本项目 Go 主栈 | 将“指纹模板生成”定义为独立领域服务；MVP 用小型受控模板，后续可离线导入数据集 |
| [Apify fingerprint-suite](https://github.com/apify/fingerprint-suite) | TypeScript，生成器 + Playwright/Puppeteer 注入；Apache-2.0 | 模块化、数据关联好、易接入自动化，迭代活跃 | JS/CDP 注入可被属性描述符、原生函数、Worker/iframe、HTTP 头与 JS 值差异识别；不等同内核修改 | 借鉴模块拆分和一致性约束；不把 JS 注入作为“强保护”默认方案 |
| [Mullvad Browser](https://mullvad.net/en/browser/hard-facts) | Tor Browser/Firefox 系隐私浏览器；开源项目 | Letterboxing 将窗口尺寸压入少量桶；清理身份、禁遥测、分级安全策略；思路透明 | 目标是群体同质化，不是任意 profile 伪装；用户改尺寸、装扩展会增加唯一性；不提供多 profile 管理面 | 默认优先“少而稳定的模板”，而不是无限随机；提供新身份/清理语义；在 UI 中解释改动的熵成本 |
| [Brave fingerprinting protections](https://github.com/brave/brave-browser/wiki/Fingerprinting-Protections) | Chromium 源码级阻断/修改 + 按站点、会话稳定随机化；MPL-2.0 | 随机值按 eTLD+1 与会话稳定，兼顾兼容性和跨站不可关联；源码级实现 | Brave 是完整浏览器而非可嵌入 SDK；随机化仍不是匿名保证；复刻需维护 Chromium fork | 未来内核策略采用“有种子、作用域明确、会话内稳定”，避免每次调用抖动 |
| [ungoogled-chromium](https://github.com/ungoogled-software/ungoogled-chromium) | Chromium 去 Google 服务与隐私补丁；BSD-3-Clause | 降低后台请求，透明、接近原生 Chromium，易作为隐私运行时 | 不是专门的指纹伪装内核；部分安全/更新能力需用户自己管理；各平台二进制来源不一 | 浏览器能力发现支持自定义 Chromium；把版本和来源显示给用户，不自动下载未知二进制 |
| [chromedp](https://github.com/chromedp/chromedp) | Go 的 Chrome DevTools Protocol 客户端；MIT | Go 原生、无需 Node driver，可做时区/语言/设备指标控制与自动化 | CDP 控制不是内核补丁；需处理所有新 target；暴露调试端口会获得高权限 | 作为未来增强适配器候选；CDP 仅监听 loopback、随机端口、生命周期内短暂存在 |
| [arkenfox user.js](https://github.com/arkenfox/user.js) | Firefox hardening 配置模板；MIT | baseline + override + 迁移/清理机制成熟；每项 preference 有风险说明 | 只是配置层，不能让任意画像在所有 API/网络层一致；过度自定义会增加唯一性 | 借鉴版本化 baseline、显式 override 和升级迁移；安全设置与身份配置分开 |
| [Patchright](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright) / [Rebrowser patches](https://github.com/rebrowser/rebrowser-patches) | Playwright driver/CDP 泄漏补丁；Apache-2.0 / MIT | 比简单改 `navigator` 更接近自动化泄漏源；可审计 patch queue | 不是浏览器内核，不解决 GPU、Canvas、字体、TLS；跟 Playwright/Chromium 版本强绑定 | 未来单独放在 Automation Adapter，不进入 Fingerprint Domain |
| [undetected-chromedriver](https://github.com/ultrafunkamsterdam/undetected-chromedriver) | Selenium/ChromeDriver 二进制与启动补丁；GPL-3.0 | 展示了驱动版本匹配、缓存和注入前处理的重要性 | 更新明显落后于浏览器；不生成完整画像；部分历史默认会降低 sandbox 安全；GPL 分发成本 | 借鉴版本检查与校验，不引入核心、不禁用 sandbox |

## 3. 商业产品为何不作为开源基线

AdsPower、Multilogin、GoLogin、Kameleo 等产品可以作为功能标杆（profile、分组、代理、团队协作、自动化 API），但其决定指纹质量的浏览器内核、真实设备数据或云服务并非完整开源。公开 SDK/示例仓库不等于核心引擎开源。因此本项目不把它们列为可直接复用的开源实现，也不接受“套一个商业二进制就算开源”的定义。

- GoLogin 的 [Orbita 文档](https://gologin.com/docs/orbita-browser) 描述了定制 Chromium，但公开的主要是控制 SDK；核心浏览器受商业条款约束。
- AdsPower 的 [Local API](https://localapi-doc-en.adspower.com/) 和 [CLI/MCP 仓库](https://github.com/AdsPower/adspower-browser) 可借鉴本地 agent 交互，SunBrowser/FlowerBrowser 内核本身不是开源基线。
- “wrapper 仓库使用 MIT”不能推导“下载的浏览器 binary 可修改或再分发”。例如 CloakBrowser 明确区分 wrapper 与 binary 许可。

## 3.1 开源真实性检查清单

评估一个候选项目时同时检查：

1. 是否能看到实际 Chromium/Firefox patch，而不是只有 Python/JS wrapper。
2. 仓库许可证是否覆盖所下载的二进制、指纹数据和云接口。
3. 是否能从固定上游 commit 重现构建，并提供 checksum/签名/SBOM。
4. 版本是否紧跟浏览器安全更新，而不只是 README 宣称“passes tests”。
5. 指纹质量声明是否由可复现测试支持，还是只依赖商业检测站截图。

没有 LICENSE 的公开仓库不自动获得复制、修改和再分发授权；open-core 项目也必须逐层审查许可。

## 4. 关键技术判断

### 4.1 一致性比字段数量重要

UA、Client Hints、TLS/HTTP2、JS 引擎、平台、GPU、字体、屏幕、语言、时区、IP 地理位置是一个关联图。仅修改 `navigator.userAgent` 会制造矛盾。Camoufox 也明确指出内部一致性是主要检测面，因此项目领域模型必须以“配置束”而不是独立开关为核心。

### 4.2 JS 注入只能标注为有限能力

注入适合测试网站在某些环境下的行为，但不能可靠覆盖 Worker、sandboxed iframe、网络请求头、原生对象形态和底层渲染。因此 MVP 不实现 canvas/audio/WebGL 伪造；这些字段只作为运行时能力报告，不制造虚假安全感。

### 4.3 profile 隔离是确定可交付的价值

独立的 user-data-dir 能隔离 cookies、localStorage、cache、扩展和站点权限。它不能改变硬件指纹，但能稳定解决 QA 多身份、回归测试、客户环境隔离、临时会话等需求。这是第一阶段的核心价值。

### 4.4 代理与 WebRTC/DNS 必须一起看

代理只覆盖浏览器按配置发出的流量；WebRTC UDP、DNS、扩展后台请求和浏览器自身服务可能形成旁路。MVP 支持 HTTP/SOCKS5 无认证代理和“限制非代理 UDP”策略，并在能力报告中明确这不是完整 VPN。

### 4.5 许可证与更新是产品能力

浏览器是高频安全更新的软件。AGPL、MPL、Chromium BSD 以及“wrapper 开源、binary 受限”的组合有完全不同的交付义务。项目不自动下载第三方内核；运行时来源、版本和能力必须可见。

## 5. 采用与放弃

### 采用

- Donut 的 profile 生命周期，以及 PinchTab 的 Go 控制面/运行时分离。
- BrowserForge/Apify 的关联配置思想。
- Mullvad 的少量标准桶和“降低唯一性”原则。
- Brave 的稳定种子与清晰作用域思想。
- Camoufox 的内核端口和能力声明方法。
- chromedp 的 Go 原生增强路线，但不放进零依赖 MVP。

### 暂不采用

- 自维护 Chromium/Firefox fork：首版成本和安全更新负担过高。
- 运行时 JS 全量 API 覆盖：容易产生可检测矛盾。
- 自动抓取代理 IP 定位：会引入外部数据、隐私和可用性依赖。
- 自动下载浏览器二进制：供应链与许可证风险不可接受。
- 目标网站特定绕过、验证码处理、行为拟人化：不属于合法 QA/隐私隔离核心价值。

## 6. 对实现方案的直接约束

1. 配置保存前执行领域一致性校验，错误阻止启动，警告必须在 UI 可见。
2. 未填写自定义 UA 时使用浏览器原生 UA，这是 MVP 最一致的默认值。
3. 每个 profile 独立目录；同一 profile 同时只允许一个运行实例。
4. runtime 以接口存在，系统 Chromium 只是第一个适配器。
5. 后端只监听 `127.0.0.1`，不开放远程 CDP，不通过 shell 启动进程。
6. 产品文案使用“隔离”“一致性”“隐私姿态”，不使用“百分百过检测”。
