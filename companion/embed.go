package companionassets

import "embed"

// Files 提供打包进二进制的 Claude Code marketplace 与 plugin 资源。
//
//go:embed all:.claude-plugin all:ccline-go
var Files embed.FS
