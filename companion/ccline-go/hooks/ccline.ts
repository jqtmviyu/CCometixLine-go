// 只从 Claude Code 本地事件采集数据，并写入 Go 状态行读取的 bridge。
import type { EngineInterface, Register } from 'claude-code'

type $ = EngineInterface

const VERSION = 1
const FLUSH_MS = 200
const MIN_STREAM_MS = 100

type Category = { name: string; kind: string; tokens: number }
type Context = {
  threshold?: number
  auto: boolean
  window?: number
  used?: number
  categories: Category[]
}
type Turn = {
  model?: string
  ttft_ms?: number
  tps?: number
  duration_ms?: number
  output_tokens?: number
  ended_at?: number
}
type Tool = { name: string; since: number; agent?: string }
type Agent = { id: string; type: string; status: string; description?: string }
type Chunk = { kind?: string; usage?: { output_tokens?: number; model?: string } | null }

const live = {
  dir: undefined as string | undefined,
  session: undefined as string | undefined,
  context: undefined as Context | undefined,
  turn: undefined as Turn | undefined,
  agents: [] as Agent[],
  tools: new Map<string, Tool>(),
  pending: false,
  seq: 0,
}

function expandHome(path: string, home: string | undefined): string {
  if (!home) return path
  if (path === '~') return home
  if (path.startsWith('~/') || path.startsWith('~\\')) return `${home}/${path.slice(2)}`
  return path
}

async function bridgeDir($: $): Promise<string | undefined> {
  if (live.dir !== undefined) return live.dir
  const home = (await $.env.get('HOME')) ?? (await $.env.get('USERPROFILE'))
  const configured = await $.env.get('CCLINE_BRIDGE_DIR')
  if (configured) {
    live.dir = expandHome(configured, home)
    return live.dir
  }
  const claudeConfig = await $.env.get('CLAUDE_CONFIG_DIR')
  const base = claudeConfig ? expandHome(claudeConfig, home) : home ? `${home}/.claude` : undefined
  if (base) live.dir = `${base}/ccline/bridge`
  return live.dir
}

function reset(session: string) {
  live.session = session
  live.context = undefined
  live.turn = undefined
  live.agents = []
  live.tools.clear()
}

async function follow($: $) {
  try {
    const id = await $.session.id()
    if (id && id !== live.session) reset(id)
  } catch {}
}

async function write($: $) {
  try {
    const dir = await bridgeDir($)
    live.session ??= await $.session.id()
    if (!dir || !live.session) return
    const snapshot = {
      v: VERSION,
      session: live.session,
      updated_at: await $.clock.now(),
      context: live.context,
      turn: live.turn,
      tools: [...live.tools.values()],
      agents: live.agents,
    }
    await $.fs.write(`${dir}/${live.session}.json`, JSON.stringify(snapshot))
  } catch {}
}

function flush($: $) {
  if (live.pending) return
  live.pending = true
  try {
    $.clock.after(FLUSH_MS, () => {
      live.pending = false
      void write($)
    })
  } catch {
    live.pending = false
  }
}

async function measure($: $) {
  try {
    const usage = await $.session.usage({ breakdown: 'summary' })
    const breakdown = usage.context.breakdown
    if (!breakdown) return
    live.context = {
      threshold: breakdown.autoCompactThreshold,
      auto: breakdown.isAutoCompactEnabled,
      window: breakdown.rawMaxTokens,
      used: breakdown.totalTokens,
      categories: breakdown.categories.map((category) => ({
        name: category.name,
        kind: category.kind,
        tokens: category.tokens,
      })),
    }
  } catch {}
}

async function listAgents($: $) {
  try {
    live.agents = (await $.agent.list()).map((agent) => ({
      id: agent.id,
      type: agent.type,
      status: agent.status,
      description: agent.description,
    }))
  } catch {}
}

async function now($: $): Promise<number> {
  return $.clock.now()
}

function recordTurn(sent: number, first: number, last: number, output?: number, model?: string) {
  const streamed = last - first
  live.turn = {
    ...live.turn,
    model: model ?? live.turn?.model,
    ttft_ms: first - sent,
    tps: output && streamed >= MIN_STREAM_MS ? (output * 1000) / streamed : live.turn?.tps,
    output_tokens: output,
    ended_at: last,
  }
}

export const register: Register = (on) => {
  live.dir = undefined
  live.pending = false

  on('session.start', async ($, event, next) => {
    const started = await next(event)
    await follow($)
    await measure($)
    await write($)
    return started
  })

  on('session.measure', async ($, event, next) => {
    const echoed = await next(event)
    await follow($)
    if (event.changed.includes('context')) {
      await measure($)
      flush($)
    }
    return echoed
  })

  // 只计主循环请求；TPS 从首个流片段开始计，不含 TTFT。
  on('turn.step', async function* ($, event, next) {
    if (event.agentId !== undefined) return yield* next(event)
    await follow($)
    const sent = await now($)
    let first: number | undefined
    let output: number | undefined
    let model: string | undefined
    const stream = next(event)
    let ended = false
    try {
      for (;;) {
        const piece = await stream.next()
        if (piece.done) {
          ended = true
          if (first !== undefined) {
            const usage = piece.value?.usage
            recordTurn(sent, first, await now($), output ?? usage?.output_tokens, model ?? usage?.model)
            flush($)
          }
          return piece.value
        }
        const chunk = piece.value as Chunk
        if (first === undefined) first = await now($)
        if (chunk.kind === 'stop' && chunk.usage) {
          output = chunk.usage.output_tokens
          model = chunk.usage.model
        }
        yield piece.value
      }
    } finally {
      if (!ended) await stream.return?.(undefined as never)
    }
  })

  on('turn.complete', async ($, event, next) => {
    const completed = await next(event)
    await follow($)
    if (event.agentId === undefined) {
      live.turn = { ...live.turn, duration_ms: event.durationMs, ended_at: await now($) }
    }
    await listAgents($)
    flush($)
    return completed
  })

  on('tool.call', async ($, event, next) => {
    const key = event.tool_use_id ?? `ccline-${live.seq++}`
    await follow($)
    live.tools.set(key, { name: event.tool, since: await now($), agent: event.agentId })
    flush($)
    try {
      return await next(event)
    } finally {
      live.tools.delete(key)
      flush($)
    }
  })

  on('agent.spawn', async ($, event, next) => {
    const spawned = await next(event)
    await follow($)
    await listAgents($)
    flush($)
    return spawned
  })

  on('session.end', async ($, event, next) => {
    live.tools.clear()
    live.agents = []
    await write($)
    return next(event)
  })
}
