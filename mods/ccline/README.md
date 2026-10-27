# ccline

CCometixLine 的会话内扩展：在输入框上方显示主循环的 TPS 和上下文分类。

## 安装

标准插件，通过仓库根目录的 `.claude-plugin/marketplace.json` 安装：

```text
/plugin marketplace add jqtmviyu/CCometixLine-go
/plugin install ccline@ccline-mods
```

或者 shell：

```bash
claude plugin marketplace add jqtmviyu/CCometixLine-go
claude plugin install ccline@ccline-mods
```

装好后在 `/plugin` 的已安装页启用、禁用或卸载，没有开关命令。临时隐藏可以用 band 右侧的 `[-]` 折叠。

## 显示

```text
(空行)
⚡ 74 tok/s
💿 prompt 1.3k · tools 20k · mcp 3.5k · agents 1.2k · ...
```

- `⚡` 每个模型请求结束时更新：请求进行中显示上一次请求的速率，结束换成本次
- 从第一个流片段到 stop 计时，`output_tokens / 流时长`；短于 100ms 保留上一次
- 第二行是本地估算的分类占用：prompt、tools、mcp、agents、memory、skills、message
- TPS 按请求更新；context 只在 session 启动、轮次结束、compact、以及 session.measure 报告 context 变化时刷新
- `mcp` 合并 MCP 工具 schema 和 server instructions
- 只列 used 且有 token 的分类

## 边界

- 单行，超宽截断
- 估算值，不是精确 token 计数；Messages 对齐上一次 API 响应
- 子代理不参与；headless 和云端不绘制

## 开发

源码在 `mods/ccline`。开发时可以只加载这个目录：

```bash
claude --plugin-dir ./mods/ccline
```

校验与测试：

```bash
claude plugin validate ./mods/ccline
claude plugin test ./mods/ccline
```
