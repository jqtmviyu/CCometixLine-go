# CCometixLine Go

使用 go 重新实现 rust 版本的 CCometixLine

## 已实现

- 读取 Claude Code stdin JSON
- 读取 `~/.claude/ccline/config.toml`
- 兼容基础配置结构
- 输出 ANSI statusline
- 已实现 segment：
  - model
  - effort (新增)
  - directory
  - git
  - context_window
  - cost
  - session
  - environment (新增)

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

## 本地运行

```bash
go run ./cmd/ccline < ./testdata/input/basic.json
```

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

- 优先级：环境变量 `CLAUDE_CODE_EFFORT_LEVEL` > 输入 JSON `effort` > `.claude/settings.local.json` > `.claude/settings.json` > `~/.claude/settings.json`
- 已知值：`low`、`medium`、`high`、`xhigh`、`max`
- `auto` 会显示为 `auto`
- 未知值如果匹配 `^[a-z0-9-]{2,20}$`，会显示为 `<value>?`
- 其他无效值会回退为 `auto`

### environment

- `mem` 统计 `CLAUDE.md / CLAUDE.local.md / rules/**/*.md`
- `mcp` 参考 claude-hud 统计已开启配置
- `skills` 优先统计已启用插件命令 + 本地 `commands/**/*.md`
- `plugins` 统计已启用插件；拿不到时显示 `?`

## 配置与默认行为

- 默认读取 `~/.claude/ccline/config.toml` 和 `~/.claude/ccline/models.toml`。
- 如果设置了 `CLAUDE_CONFIG_DIR`，`config.toml` 和 `models.toml` 都切换到 `<CLAUDE_CONFIG_DIR>/ccline/`。
- 默认配置启用 `model`、`effort`、`directory`、`git`、`context_window`、`environment`。
- 默认配置关闭 `cost`、`session`。

## 接入 Claude Code

构建后，把 Claude Code 的 `statusLine.command` 指向新可执行文件。

Windows 示例：

```json
{
  "statusLine": {
    "type": "command",
    "command": "C:\\Users\\demo\\.claude\\ccline\\ccline.exe"
  }
}
```
