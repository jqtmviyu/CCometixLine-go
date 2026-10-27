# CCometixLine Go

参考上游项目 [Haleclipse/CCometixLine](https://github.com/Haleclipse/CCometixLine)，使用 Go 重新实现，并针对精简使用和第三方模型做了定制。

## 已实现

- 读取 Claude Code statusline stdin JSON，并按官方输入 token 口径展示上下文占用
- 支持缓存状态、缓存命中率、miss 原因；冷缓存成本只在模型价格和 TTL 写入倍率已配置时显示
- 支持 `subagentStatusLine` 的任务名称、状态、label、耗时和 token 趋势
- 支持显式固定行布局：默认两行，可按 segment 自定义字段和断点，不按宽度自动折行
- 输出 ANSI statusline
- segment：model、effort、directory、git、context_window、cache、cache.cold_cost、cost、session、environment 和 task.*

## 未实现
- `--config` TUI
- `--patch`
- Update segment
- 主题文件自动生成
- 未实现 segment：
  - usage: 需要 A/ 官方api 才支持, 为了缩小包体积直接干掉
  - output_style

## 目录

- `cmd/ccline/main.go`：CLI 入口
- `internal/config`：配置、路径、模型配置
- `internal/protocol`：Claude Code 输入协议、usage 归一化
- `internal/render`：ANSI 与 statusline 渲染
- `internal/segments`：各 segment 采集
- `testdata`：样例输入与 transcript
- `examples/config.toml`：示例配置

## 本地运行与预览

Claude Code 管道输入时，程序按 statusline 协议读取 stdin：

```bash
go run ./cmd/ccline < ./testdata/input/basic.json
```

在终端里直接运行 `ccline` 会展示当前配置的示例预览；也可以显式运行：

```bash
go run ./cmd/ccline preview
```

预览使用固定示例数据，不读取 Git 状态、项目 settings、本地缓存，也不会修改配置。它展示当前启用 segment 的图标、颜色和显式行布局；示例值不代表实时会话。

## 测试

```bash
go test ./...
```

## 构建

```bash
./build.sh darwin-amd64
./build.sh darwin-arm64
./build.sh darwin-universal
./build.sh windows-amd64
./build.sh windows-arm64
./build.sh all
```

产物：
- `build/ccline_darwin_amd64`
- `build/ccline_darwin_arm64`
- `build/ccline_darwin_universal`
- `build/ccline_windows_amd64.exe`
- `build/ccline_windows_arm64.exe`

## 新增功能

### effort

- 显示优先级：环境变量 `CLAUDE_CODE_EFFORT_LEVEL` > 实时输入 JSON `effort.level` > 设置文件 `effortLevel`
- 设置文件依次读取：`~/.claude/settings.json`、`<cwd>/.claude/settings.json`、`<cwd>/.claude/settings.local.json`
- 已知值：`auto`、`low`、`medium`、`high`、`xhigh`、`max`
- 未知但格式合法的值显示为 `value?`
- 非法值忽略，继续回退到更低优先级来源
- 默认值：`auto`

### environment

- `mem` 统计 `CLAUDE.md / CLAUDE.local.md / rules/**/*.md`
- `mcp` 统计 `~/.claude/settings.json`、`<cwd>/.claude/settings.json`、`<cwd>/.claude/settings.local.json` 里的 `enabledMcpjsonServers`；按名称去重
- `skills` 统计 `~/.claude/skills/*/SKILL.md` 与 `<cwd>/.claude/skills/*/SKILL.md`；过滤 YAML 头部中 `disable-model-invocation: true` 的项，按子目录名称去重。只读取头部，正文不参与判断。
- skill 子目录支持指向目录的软链接，适用于共享安装的 skills。
- `plugins` 统计 `~/.claude/settings.json`、`<cwd>/.claude/settings.json`、`<cwd>/.claude/settings.local.json` 里的 `enabledPlugins=true`；按名称去重
- 状态栏将环境计数显示为 `memory 3 · skill 5 · mcp 2 · plugin 1`。
- `mem/skills/mcp/plugins` 汇总按配置目录与当前工作目录缓存 60 秒，缓存只保存计数；缓存写入独立的本地缓存文件。缓存到期后的首次刷新重新统计，可通过 `[cache]` 调整时长。

## 配置与默认行为

- 默认读取 `~/.claude/ccline/config.toml` 和 `~/.claude/ccline/models.toml`；设置 `CLAUDE_CONFIG_DIR` 后改为 `<CLAUDE_CONFIG_DIR>/ccline/`。
- `config.toml` 用 `[[layout.lines]]` 固定分组和显示顺序；默认两行，添加、删除或合并分组即可配置一行、三行或任意自定义断点。没有可显示内容的组会省略，因此实际输出可能少于两行。完整样例见 `examples/config.toml`。
- 每组只显示列出的 segment ID；想隐藏字段就从布局中移除，或在 `[[segments]]` 中设为 `enabled = false`。主 statusline 不会按长度自动折行，长行可通过选择字段或增加显式行组控制。运行 `ccline preview` 查看效果。
- 显示顺序由 layout 决定；不写 `[[segments]]` 时使用内置 segment 默认值。若显式写 `[[segments]]`，列表中的项目同时定义启用状态、图标、颜色和选项，未列出的 segment 不参与渲染。
- 旧版 Go `[[segments]]` 配置仍可读取，且默认按原顺序排成一行；上游 CCometixLine 的配置格式不兼容。
- 默认第一行集中模型/力度、工作区/Git，最后显示 tokens；第二行从 memory/skills/MCP/plugins 开始，随后显示缓存、费用/会话。`cost`、`session` 默认关闭。可按需调整布局和启用字段。
- 旧配置中的 `style.wrap_width` 会被忽略；主 statusline 不做宽度检测或自动折行。
- `effort` 读取用户级设置时跟随 `CLAUDE_CONFIG_DIR`。
- 状态栏只使用一套 emoji 图标，不提供 Nerd Font 档位。默认图标：模型 `🤖`、Git `🐙`、environment `🧰`、tokens `💿`、warm cache `🔥`、cold cost `💲`；effort 无图标，并与模型相邻显示为 `模型 · 思考力度`，思考力度和中间的 `·` 沿用模型配色及文字样式。
- `task.status` 使用 `🏃` 表示 running、`✓` 表示 completed、`✗` 表示 failed。
- 自定义 segment 图标使用 `icon.emoji`：

```toml
[[segments]]
id = "model"
enabled = true

[segments.icon]
emoji = "🤖"
```

缓存成本需要在 `models.toml` 中为模型配置输入价格和对应 TTL 的缓存写入倍率；没有匹配项时只显示缓存状态，不猜价格：

```toml
[[models]]
pattern = "my-provider/model-x"
match = "prefix" # contains (默认)、prefix、exact、regex
input_price_usd_per_million = 2.5
cache_write_multipliers = { "5m" = 1.25, "1h" = 2.0 }
```

## 采集开销与缓存

状态栏刷新与数据采集分开控制。当前目录直接来自 stdin；Git 默认按工作目录缓存 3 秒，memory/skills/MCP/plugins 计数默认缓存 60 秒。到期后的首次刷新同步采集，期间显示上次结果，因此更新延迟还取决于 Claude Code 的刷新间隔。

在 `config.toml` 中配置缓存时长，单位为秒；`0` 表示每次重新采集：

```toml
[cache]
git_seconds = 3
environment_seconds = 60
```

缓存默认保存到 `~/.claude/ccline/cache/` 下的 `git/` 和 `environment/` 子目录；设置 `CLAUDE_CONFIG_DIR` 后使用 `<CLAUDE_CONFIG_DIR>/ccline/cache/`，也可以通过 `CCLINE_CACHE_DIR` 单独指定缓存根目录。Git 缓存也保存“当前目录没有仓库”的结果，新建仓库后在到期刷新时显示；切换目录使用各自的缓存。缓存采用完整快照替换，无需同步落盘，写入失败时继续输出本次采集结果。

从布局移除 `git`、`environment` 或禁用对应 segment，会跳过其采集。effort 已由有效环境变量或 stdin 确定时，跳过为 effort 读取 settings；环境计数缓存过期时仍需读取 settings。

## 缓存与子任务输入

状态栏直接使用 Claude Code 提供的 stdin JSON，无需额外安装插件。

- `cache` 显示 Claude Code 提供的会话缓存状态、命中率、miss 数和最后 miss 原因；命中率沿用 Claude Code 的输入 token 统计，不从单次缓存 token 估算。缺少缓存输入时隐藏。
- `cache.cold_cost` 估算公式为 `recache_tokens_if_cold × input_price_usd_per_million × cache_write_multiplier ÷ 1,000,000`，缺少任一价格参数时隐藏。

`task.*` 通过 Claude Code 的 `subagentStatusLine` 输入工作，需要单独把该设置也指向同一个可执行文件；主 statusline 的普通文本协议与 subagent 的 JSONL 输出协议会根据 stdin 自动区分。

## 接入 Claude Code

构建后，把 Claude Code 的 `statusLine.command` 指向新可执行文件。

Windows 示例：

```json
{
  "statusLine": {
    "type": "command",
    "command": "C:\\Users\\demo\\.claude\\ccline\\ccline.exe",
    "refreshInterval": 2
  },
  "subagentStatusLine": {
    "type": "command",
    "command": "C:\\Users\\demo\\.claude\\ccline\\ccline.exe"
  }
}
```

## 从带 companion 的旧版升级

新版已移除 companion、`ccline mod` 命令和依赖插件的上下文分类、TTFT/TPS、工具及 agent 活动字段。旧配置中的这些字段会被忽略，其余字段继续按原布局显示。

曾安装旧版插件时，可通过 Claude Code 卸载，并在当前会话执行 `/reload-plugins`：

```bash
claude plugin uninstall ccline-go@ccometixline-go --scope user
```

旧版的 `ccline/bridge/`、`ccline/mod/` 文件不会再被读取，也不会自动删除。新版会在独立缓存目录中重建 Git 和环境计数缓存；自定义缓存路径使用 `CCLINE_CACHE_DIR`。
