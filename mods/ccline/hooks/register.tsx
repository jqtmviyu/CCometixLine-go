import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

const MIN_STREAM_MS = 100
const GOLD = '#f5c542'
const RATE = '#7dcfff'
const CONTEXT = '#bb9af7'
const ORDER = ['prompt', 'tools', 'mcp', 'agents', 'memory', 'skills', 'message']
const LABELS: Record<string, string> = {
  'system prompt': 'prompt',
  'system tools': 'tools',
  'mcp tools': 'mcp',
  'mcp tools (deferred)': 'mcp',
  'mcp server instructions': 'mcp',
  'mcp instructions': 'mcp',
  'custom agents': 'agents',
  agents: 'agents',
  'memory files': 'memory',
  memory: 'memory',
  skills: 'skills',
  messages: 'message',
}

type ContextItem = { label: string; tokens: number }

// 速率放在 $.state：热重载后仍保留上一次的值，/clear 时归零。
const rateAtom = atom({ plugin: 'ccline', key: 'rate' } as const, null as number | null)

let reading: ContextItem[] | null = null

function formatRate(value: number): string {
  return value >= 10 ? `${Math.round(value)}` : value.toFixed(1)
}

function formatTokens(value: number): string {
  if (value >= 1_000_000) return `${+(value / 1_000_000).toFixed(value % 1_000_000 === 0 ? 0 : 1)}M`
  if (value >= 10_000) return `${Math.round(value / 1000)}k`
  if (value >= 1000) return `${+(value / 1000).toFixed(1)}k`
  return String(value)
}

function labelFor(name: string): string {
  return LABELS[name.toLowerCase()] ?? name.toLowerCase()
}

function rank(label: string): number {
  const index = ORDER.indexOf(label)
  return index < 0 ? ORDER.length : index
}

// summary 是本地估算，不调用 token-count API；够看分类结构。
async function refresh($: EngineInterface) {
  try {
    const usage = await $.session.usage({ breakdown: 'summary' })
    const categories = usage.context.breakdown?.categories ?? []
    const totals = new Map<string, number>()
    for (const category of categories) {
      if (category.kind !== 'used' || category.tokens <= 0) continue
      const label = labelFor(category.name)
      totals.set(label, (totals.get(label) ?? 0) + category.tokens)
    }
    reading = [...totals.entries()]
      .map(([label, tokens]) => ({ label, tokens }))
      .sort((a, b) => rank(a.label) - rank(b.label))
  } catch {
    reading = null
  }
}

export const register: Register = (on) => {
  on('session.start', async ($, e, next) => {
    $.ui.status(undefined) // 清掉旧版本留下的 status 行
    await refresh($)
    return next(e)
  })

  on('session.end', async ($, e, next) => {
    if (e.reason === 'clear' || e.reason === 'resume') {
      await update($, rateAtom, () => null)
      reading = null
      $.ui.status(undefined)
      $.ui.invalidate('ui.render')
    }
    return next(e)
  })

  on('session.compact', async ($, e, next) => {
    const result = await next(e)
    await refresh($)
    $.ui.invalidate('ui.render')
    return result
  })

  // 上下文比 TPS 更新得慢：只有引擎说 context 变了才重新取 breakdown。
  on('session.measure', async ($, e, next) => {
    const result = await next(e)
    if (e.changed.includes('context')) {
      await refresh($)
      $.ui.invalidate('ui.render')
    }
    return result
  })

  // 照上游 ccline：每个主循环请求结束时更新速率，不等整轮结束。
  // 从第一个流片段（含信封）到 stop 计时，output 取 stop 的 usage。
  on('turn.step', async function* ($, e, next) {
    if (e.agentId !== undefined) return yield* next(e)

    let first: number | undefined
    let last = await $.clock.now()
    let output: number | undefined
    const stream = next(e)
    let done = false

    try {
      for (;;) {
        const piece = await stream.next()
        last = await $.clock.now()
        if (piece.done) {
          done = true
          const tokens = output ?? piece.value.usage?.output_tokens ?? 0
          const started = first
          if (started !== undefined && tokens > 0 && last - started >= MIN_STREAM_MS) {
            const measured = (tokens * 1000) / (last - started)
            await update($, rateAtom, () => measured)
          }
          return piece.value
        }

        const chunk = piece.value
        if (first === undefined) first = last
        if (chunk.kind === 'stop' && chunk.usage) output = chunk.usage.output_tokens
        yield chunk
      }
    } finally {
      if (!done) await stream.return?.(undefined as never)
    }
  })

  on('turn.complete', async ($, e, next) => {
    if (e.agentId === undefined) {
      await refresh($)
      $.ui.invalidate('ui.render')
    }
    return next(e)
  })

  // 单行 band：没有 status 的 ⚠ 前缀，颜色也压暗；输入框上边距是引擎的。
  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => {
    if (e.props.hasSurvey) return next(e)
    const currentRate = await read($, rateAtom)
    if (currentRate === null && (reading === null || reading.length === 0)) return next(e)

    const { Box, Text } = $.ui.resolve(e)

    return (
      <Box flexDirection="column">
        <Text> </Text>
        {currentRate !== null && (
          <Box flexDirection="row" columnGap={1}>
            <Text color={GOLD}>⚡</Text>
            <Text color={RATE}>{formatRate(currentRate)} tok/s</Text>
          </Box>
        )}
        {reading !== null && reading.length > 0 && (
          <Box flexDirection="row" columnGap={1}>
            <Text color={CONTEXT}>💿</Text>
            <Text wrap="truncate-end">
              {reading.map((item, i) => (
                <Text key={item.label}>
                  {i > 0 && <Text dimColor>{' · '}</Text>}
                  <Text dimColor>{`${item.label} `}</Text>
                  <Text>{formatTokens(item.tokens)}</Text>
                </Text>
              ))}
            </Text>
          </Box>
        )}
      </Box>
    )
  })
}
