import type { BatchTypeGroup, ContentTab, Entry, EntryDetailResponse, EntryType, ListResult, SignalSetting, StorageUsage } from './types'

const now = Date.now()
const languages = ['go', 'python', 'node', 'php', 'ruby', 'elixir']
const frameworks = ['net/http', 'FastAPI', 'NestJS', 'Laravel', 'Rails', 'Phoenix']
const types: EntryType[] = ['request', 'query', 'redis', 'job', 'event', 'log', 'exception', 'http-client', 'performance', 'metric', 'topic', 'cache']
const paths = ['/api/checkout', '/v1/orders', '/auth/session', '/api/catalog', '/webhooks/payment']

function entry(index: number): Entry {
  const type = types[index % types.length]
  const language = languages[index % languages.length]
  const framework = frameworks[index % frameworks.length]
  const duration = [18, 42, 76, 128, 245, 640, 91][index % 7]
  const batch = `demo-batch-${Math.floor(index / 4)}`
  const base = {
    id: `demo-${String(index + 1).padStart(4, '0')}-signal`,
    batch_id: batch,
    request_id: `req-${String(Math.floor(index / 4) + 1).padStart(4, '0')}`,
    correlation_id: `corr-checkout-${Math.floor(index / 8) + 1}`,
    tags: [index % 2 ? 'production-shadow' : 'local', framework.toLowerCase().replace('/', '-')],
    type,
    created_at: new Date(now - index * 41_000).toISOString(),
    content: {} as Record<string, unknown>,
  }
  const content: Record<EntryType, Record<string, unknown>> = {
    request: { method: index % 3 ? 'GET' : 'POST', path: paths[index % paths.length], status: index === 24 ? 503 : 200, duration_ms: duration, memory_mb: 48 + index },
    query: { sql: 'SELECT orders.id, orders.total, users.email FROM orders INNER JOIN users ON users.id = orders.user_id WHERE orders.status = $1 AND orders.created_at > $2 ORDER BY orders.created_at DESC LIMIT 50', bindings: ['processing', '2026-07-27T00:00:00Z'], duration_ms: duration, connection: 'postgres-primary' },
    redis: { command: 'GET', key: `session:${index}`, duration_ms: duration },
    job: { name: 'SendOrderReceipt', queue: 'critical', attempts: 1, duration_ms: duration },
    event: { name: 'checkout.completed', payload: { order_id: 8400 + index, total: 129.95, currency: 'USD', customer: { tier: 'pro', region: 'eu-west' } } },
    log: { level: index % 5 ? 'info' : 'warning', message: 'Payment gateway response normalized', context: { provider: 'stripe', retry: false, latency_ms: duration } },
    exception: { kind: 'InventoryReservationError', message: 'Inventory changed while the order was being committed', stack: 'inventory.reserve (inventory.go:184)\ncheckout.commit (checkout.go:92)' },
    schedule: { name: 'reconcile-orders', duration_ms: duration },
    mail: { subject: 'Your order is confirmed', to: 'developer@example.test' },
    notification: { channel: 'push', template: 'order-ready' },
    'http-client': { method: 'POST', url: 'https://api.stripe.test/v1/payment_intents', status: 200, duration_ms: duration },
    websocket: { channel: 'orders.live', action: 'broadcast', clients: 18 },
    performance: { name: 'checkout.pipeline', duration_ms: duration, memory_mb: 72 },
    metric: { name: 'runtime.heap.used', value: 54 + index, unit: 'MB', language, framework, duration_ms: duration },
    custom: { name: 'release.marker', version: '2026.07.27' },
    topic: { action: 'produce', topic: 'order-events', partition: index % 4, offset: 18000 + index, message_count: 1, size_bytes: 842, duration_ms: duration },
    cache: { operation: index % 2 ? 'hit' : 'miss', key: `catalog:${index}`, duration_ms: duration },
  }
  base.content = content[type]
  return base
}

export const demoEntries: Entry[] = Array.from({ length: 44 }, (_, index) => entry(index))

export function demoList(params: URLSearchParams): ListResult {
  let result = [...demoEntries]
  const type = params.get('type')
  const search = (params.get('search') || '').toLowerCase()
  if (type) result = result.filter(item => item.type === type)
  if (search) result = result.filter(item => JSON.stringify(item).toLowerCase().includes(search))
  const offset = Number(params.get('offset') || 0)
  const limit = Number(params.get('limit') || 80)
  return { entries: result.slice(offset, offset + limit), total: result.length }
}

export function demoDetail(id: string): EntryDetailResponse {
  const selected = demoEntries.find(item => item.id === id) || demoEntries[0]
  const batch = demoEntries.filter(item => item.batch_id === selected.batch_id)
  const related = batch.reduce((groups, item) => {
    const existing = groups.find(group => group.type === item.type)
    if (existing) existing.entries.push(item)
    else groups.push({ type: item.type, label: item.type.replace('-', ' '), entries: [item] })
    return groups
  }, [] as BatchTypeGroup[])

  const tabs: ContentTab[] = []
  if (selected.type === 'request') {
    const content = selected.content
    tabs.push(
      {
        id: 'payload',
        label: 'Payload',
        body: JSON.stringify({
          order_id: 8400 + demoEntries.indexOf(selected),
          items: [{ sku: 'SKU-441', qty: 2, price: 49.99 }, { sku: 'SKU-882', qty: 1, price: 29.97 }],
          customer: { email: 'developer@example.test', tier: 'pro' },
          metadata: { source: 'web-checkout', campaign: 'summer-2026' },
        }, null, 2),
        json: true,
      },
      {
        id: 'headers',
        label: 'Headers',
        body: JSON.stringify({
          'content-type': 'application/json',
          authorization: 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.demo',
          'x-request-id': selected.request_id,
          'user-agent': 'MicroscopeDemo/1.0',
          accept: 'application/json',
          cookie: 'session=demo-session-token; preferences=dark',
        }, null, 2),
        json: true,
      },
      {
        id: 'response',
        label: 'Response',
        body: JSON.stringify({
          status: content.status || 200,
          message: Number(content.status) >= 500 ? 'Service unavailable' : 'Order accepted',
          order: { id: 8400 + demoEntries.indexOf(selected), total: 129.95, currency: 'USD' },
          timing: { duration_ms: content.duration_ms, processed_at: selected.created_at },
        }, null, 2),
        json: true,
      },
    )
  } else {
    tabs.push(
      { id: 'content', label: 'Content', body: JSON.stringify(selected.content, null, 2), json: true },
      { id: 'metadata', label: 'Metadata', body: JSON.stringify({ tags: selected.tags, request_id: selected.request_id, correlation_id: selected.correlation_id, captured_by: 'standalone-demo' }, null, 2), json: true },
    )
  }

  return { entry: selected, batch, batch_groups: related, content_tabs: tabs, related_active_tab: selected.type }
}

export function demoSettings(): SignalSetting[] {
  return types.map(type => ({ type, enabled: true, count: demoEntries.filter(entry => entry.type === type).length }))
}

export function demoStorageUsage(): StorageUsage {
  const entries = demoEntries.length
  return {
    entries_mb: 4.28,
    entries_data_mb: 3.42,
    entries_indexes_mb: 0.86,
    settings_mb: 0.01,
    migrations_mb: 0.03,
    total_mb: 4.32,
    entry_count: entries,
  }
}

