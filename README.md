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

- 显示优先级：环境变量 `CLAUDE_CODE_EFFORT_LEVEL` > 设置文件 `effortLevel` > 实时输入 JSON `effort.level`
- 设置文件依次读取：`~/.claude/settings.json`、`<cwd>/.claude/settings.json`、`<cwd>/.claude/settings.local.json`
- 已知值：`auto`、`low`、`medium`、`high`、`xhigh`、`max`
- 未知但格式合法的值显示为 `value?`
- 非法值忽略，继续回退到更低优先级来源
- 默认值：`auto`

### environment

- `mem` 统计 `CLAUDE.md / CLAUDE.local.md / rules/**/*.md`
- `mcp` 统计 `~/.claude/settings.json`、`<cwd>/.claude/settings.json`、`<cwd>/.claude/settings.local.json` 里的 `enabledMcpjsonServers`；按名称去重
- `skills` 统计 `~/.claude/skills` 与 `<cwd>/.claude/skills` 下的 skill 子目录名称总数；同名目录去重
- `plugins` 统计 `~/.claude/settings.json`、`<cwd>/.claude/settings.json`、`<cwd>/.claude/settings.local.json` 里的 `enabledPlugins=true`；按名称去重

## 配置与默认行为

- 默认读取 `~/.claude/ccline/config.toml` 和 `~/.claude/ccline/models.toml`。
- 如果设置了 `CLAUDE_CONFIG_DIR`，`config.toml` 和 `models.toml` 都切换到 `<CLAUDE_CONFIG_DIR>/ccline/`。
- `effort` 读取用户级设置时也会跟随 `CLAUDE_CONFIG_DIR`，即用户设置文件位置变为 `<CLAUDE_CONFIG_DIR>/settings.json`。
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
