import type { On } from 'claude-code'
import type { Engine } from 'claude-code/testing'
import { expect, mock, test } from 'claude-code/testing'

const USAGE = {
  input_tokens: 10,
  output_tokens: 50,
  cache_read_input_tokens: 0,
  cache_creation_input_tokens: 0,
  model: 'test-model',
}

const CATEGORIES = [
  { name: 'System prompt', tokens: 3400, kind: 'used', isDeferred: false, color: 'promptBorder' },
  { name: 'System tools', tokens: 12000, kind: 'used', isDeferred: false, color: 'inactive' },
  { name: 'MCP tools', tokens: 2600, kind: 'used', isDeferred: false, color: 'cyan' },
  { name: 'MCP server instructions', tokens: 900, kind: 'used', isDeferred: false, color: 'cyan' },
  { name: 'Custom agents', tokens: 1200, kind: 'used', isDeferred: false, color: 'purple' },
  { name: 'Memory files', tokens: 800, kind: 'used', isDeferred: false, color: 'warning' },
  { name: 'Skills', tokens: 500, kind: 'used', isDeferred: false, color: 'green' },
  { name: 'Messages', tokens: 186000, kind: 'used', isDeferred: false, color: 'red' },
  { name: 'Free space', tokens: 746000, kind: 'free', isDeferred: false, color: 'promptBorder' },
  { name: 'Autocompact buffer', tokens: 50000, kind: 'buffer', isDeferred: false, color: 'inactive' },
]

const ABOVE = {
  plugin: 'ccline',
  component: 'AbovePrompt',
  props: {
    hasSurvey: false,
    isWorking: true,
    maxRows: 3,
    bodyColumns: 80,
    scroll: { offset: 0, bodyRows: 3 },
    view: {},
  },
} as const

function engineDraws(on: On) {
  let calls = 0
  on('session.usage', (() => {
    calls += 1
    return {
      value: {
        startedAt: 0,
        context: {
          tokens: 204000,
          window: 1000000,
          percent: 20,
          breakdown: { categories: CATEGORIES },
        },
        rateLimits: [],
        cost: { usd: 0 },
      },
    }
  }) as any)
  on('ui.render', ($, e) => {
    const { Text } = $.ui.resolve(e)
    if (e.component === 'AbovePrompt') return <Text dimColor>other band</Text>
    return <Text> </Text>
  })
  on('session.measure', (() => ({ changed: ['context'] })) as any)
  return { calls: () => calls }
}

// 一次 500ms 的请求；getOutput 决定这次的输出 token 数。
function streamHooks(on: On, clock: ReturnType<typeof mock.clock>, getOutput: () => number) {
  on('turn.start', (_, e) => ({ turnId: e.turnId }))
  on('turn.step', async function* (_, e) {
    yield { kind: 'text', index: 0, text: 'hello' }
    await clock.advance(500)
    const usage = { ...USAGE, output_tokens: getOutput() }
    return {
      turnId: e.turnId,
      index: e.index,
      answer: 'hello',
      toolUses: [],
      stopReason: 'end_turn',
      usage,
    }
  })
  on('turn.complete', (_, e) => ({ text: e.answer }))
}

async function runTurn($: Engine, turnId: string, output: number) {
  await $.turn.start({ text: 'hi', turnId })
  const stream = $.turn.step({ turnId, index: 0, model: 'test-model', messageCount: 1 })
  for (;;) {
    const piece = await stream.next()
    if (piece.done) break
  }
  await $.turn.complete({
    answer: 'hello',
    durationMs: 500,
    isAborted: false,
    turnId,
    reason: 'answer',
    usage: { ...USAGE, output_tokens: output },
  })
}

test('band 显示速率和 context，无 status 前缀', async ($, on) => {
  const clock = mock.clock(on)
  engineDraws(on)
  streamHooks(on, clock, () => 50)
  await runTurn($, 't1', 50)

  const ui = await $.ui.mount({ ...ABOVE, surface: 'terminal' })
  expect((await ui.find({ type: 'Text', text: /100 tok\/s/ }))?.text).toContain('100 tok/s')
  expect(await ui.find({ type: 'Text', text: /💿/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /mcp 3\.5k/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /message 186k/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /⚠/ })).toBeUndefined()
  await ui.unmount()
})

test('本轮进行中先显示上一轮速率，请求结束后替换', async ($, on) => {
  const clock = mock.clock(on)
  engineDraws(on)
  let output = 50
  streamHooks(on, clock, () => output)
  await runTurn($, 'prev', 50)

  output = 20
  await $.turn.start({ text: 'hi', turnId: 'next' })
  const stream = $.turn.step({ turnId: 'next', index: 0, model: 'test-model', messageCount: 1 })
  await stream.next()

  const during = await $.ui.mount({ ...ABOVE, surface: 'terminal' })
  expect((await during.find({ type: 'Text', text: /100 tok\/s/ }))?.text).toBeDefined()
  expect((await during.find({ type: 'Text', text: /40 tok\/s/ }))?.text).toBeUndefined()
  await during.unmount()

  for (;;) {
    const piece = await stream.next()
    if (piece.done) break
  }

  const after = await $.ui.mount({ ...ABOVE, surface: 'terminal' })
  expect((await after.find({ type: 'Text', text: /40 tok\/s/ }))?.text).toBeDefined()
  await after.unmount()
})

test('第一个请求结束后就更新速率，不等整轮结束', async ($, on) => {
  const clock = mock.clock(on)
  engineDraws(on)
  let output = 50
  streamHooks(on, clock, () => output)

  await $.turn.start({ text: 'hi', turnId: 'multi' })
  const first = $.turn.step({ turnId: 'multi', index: 0, model: 'test-model', messageCount: 1 })
  for (;;) {
    const piece = await first.next()
    if (piece.done) break
  }

  output = 20
  const second = $.turn.step({ turnId: 'multi', index: 1, model: 'test-model', messageCount: 2 })
  await second.next()

  const mid = await $.ui.mount({ ...ABOVE, surface: 'terminal' })
  expect((await mid.find({ type: 'Text', text: /100 tok\/s/ }))?.text).toBeDefined()
  await mid.unmount()

  for (;;) {
    const piece = await second.next()
    if (piece.done) break
  }

  const after = await $.ui.mount({ ...ABOVE, surface: 'terminal' })
  expect((await after.find({ type: 'Text', text: /40 tok\/s/ }))?.text).toBeDefined()
  await after.unmount()

  await $.turn.complete({
    answer: 'hello',
    durationMs: 1000,
    isAborted: false,
    turnId: 'multi',
    reason: 'answer',
    usage: { ...USAGE, output_tokens: 20 },
  })
})

test('子代理请求不替换速率', async ($, on) => {
  const clock = mock.clock(on)
  engineDraws(on)
  let output = 50
  streamHooks(on, clock, () => output)
  await runTurn($, 'main-sub', 50)

  output = 20
  await $.turn.start({ text: 'hi', turnId: 'a1', agentId: 'a1' })
  const stream = $.turn.step({ turnId: 'a1', index: 0, model: 'test-model', messageCount: 1, agentId: 'a1' })
  for (;;) {
    const piece = await stream.next()
    if (piece.done) break
  }

  const ui = await $.ui.mount({ ...ABOVE, surface: 'terminal' })
  expect((await ui.find({ type: 'Text', text: /100 tok\/s/ }))?.text).toBeDefined()
  expect((await ui.find({ type: 'Text', text: /40 tok\/s/ }))?.text).toBeUndefined()
  await ui.unmount()
})

test('session.measure 变更 context 时刷新', async ($, on) => {
  const clock = mock.clock(on)
  const { calls } = engineDraws(on)
  streamHooks(on, clock, () => 50)
  await runTurn($, 'measure', 50)

  const before = calls()
  await $.session.measure({ changed: ['context'] } as any)
  expect(calls()).toBeGreaterThan(before)
})
