# 升级、备份与回滚

Profile 元数据和浏览器 user-data 目录都是本地重要数据。升级前必须做离线备份；
只复制 `profiles.json` 而遗漏 `browser-data` 会丢失 cookies、localStorage、缓存和
站点权限等会话数据。

## 找到数据目录

显式设置 `PROFILEWEAVE_DATA_DIR` 时，以该路径为准。早期版本可能使用
`FINGERPRINT_BROWSER_DATA_DIR`。未设置时，默认位置通常是操作系统用户配置目录
下的 `ProfileWeave`：

- Windows：`%APPDATA%\ProfileWeave`
- macOS：`~/Library/Application Support/ProfileWeave`
- Linux：`$XDG_CONFIG_HOME/ProfileWeave`，未设置 XDG 时通常是
  `~/.config/ProfileWeave`

升级前应通过当前版本 README、启动输出和实际文件位置再次确认，不要根据示例路径
盲目覆盖目录。有效数据目录至少包含 `profiles.json`，有已启动 Profile 时还包含
`browser-data/`。

## 创建一致备份

1. 在管理台停止所有正在运行的 Profile。
2. 退出 ProfileWeave 服务端，并确认由它启动的浏览器进程已经退出。
3. 将整个数据目录复制到新的、访问受限的备份位置。
4. 检查备份是否包含 `profiles.json` 与完整的 `browser-data/`。
5. 记录来源版本、操作系统、浏览器版本和备份时间。

Windows 示例（将路径替换为已经核实的实际路径）：

```powershell
$source = Join-Path $env:APPDATA 'ProfileWeave'
$destination = Join-Path $env:USERPROFILE 'ProfileWeave-backup-2026-07-17.zip'
Compress-Archive -LiteralPath $source -DestinationPath $destination
Get-FileHash -LiteralPath $destination -Algorithm SHA256
```

浏览器 Profile 中可能存在登录态、访问令牌、历史记录和个人数据。备份应使用受限
权限和可信的加密存储，不应上传到公开 issue、普通聊天或未受控云盘。

## 升级步骤

1. 阅读 `CHANGELOG.md` 和目标版本的 Release Notes，特别关注存储 schema、浏览器
   参数和环境变量变化。
2. 完成离线备份并记录哈希。
3. 下载新归档，按 `docs/installation.md` 校验 SHA-256 和 provenance。
4. 解压到新目录；不要直接覆盖仍在运行的旧二进制和前端文件。
5. 让新旧版本指向同一个已核实的数据目录，但一次只能启动一个版本。
6. 启动新版本，检查 Profile 数量、配置诊断和浏览器发现结果。
7. 用非敏感测试 Profile 验证启动、停止和数据保留，再继续日常使用。

存储格式变更必须由应用提供显式迁移并保持原子写入。若新版本报告不支持的数据
schema，应立即退出，不要手工修改 JSON 猜测字段。

## 回滚与恢复

二进制回滚不等于数据回滚。新版本一旦迁移或写入数据，旧版本可能无法读取。
安全回滚流程是：

1. 停止所有 Profile 和 ProfileWeave。
2. 保留故障现场副本，避免覆盖唯一可诊断数据。
3. 将活动数据目录移动到另一个明确命名的位置。
4. 从升级前的完整备份恢复原目录。
5. 启动与该备份版本一致的旧发行版并验证。

不要在应用或浏览器仍运行时恢复，不要把来自不可信来源的 browser-data 目录导入
工作环境。恢复完成后，限制或安全处置包含旧会话数据的临时副本。
