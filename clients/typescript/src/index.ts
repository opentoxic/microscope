export interface MicroscopeClientOptions {
  baseUrl: string;
  fetchImpl?: typeof fetch;
}

export interface ListEntriesQuery {
  type?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export class MicroscopeClient {
  private readonly baseUrl: string;
  private readonly fetchImpl: typeof fetch;
  private metricsTimer: ReturnType<typeof setInterval> | null = null;

  constructor(options: MicroscopeClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/$/, "");
    this.fetchImpl = options.fetchImpl ?? fetch;
  }

  async record(name: string, content: Record<string, unknown> = {}): Promise<string> {
    const res = await this.fetchImpl(`${this.baseUrl}/api/entries`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ name, content }),
    });
    if (!res.ok) {
      throw new Error(`microscope: record failed with status ${res.status}`);
    }
    const body = (await res.json()) as { id: string };
    return body.id;
  }

  async listEntries(query: ListEntriesQuery = {}): Promise<unknown> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) params.set(key, String(value));
    }
    const res = await this.fetchImpl(`${this.baseUrl}/api/entries?${params}`);
    if (!res.ok) {
      throw new Error(`microscope: listEntries failed with status ${res.status}`);
    }
    return res.json();
  }

  async getEntry(entryId: string): Promise<unknown> {
    const res = await this.fetchImpl(`${this.baseUrl}/api/entries/${entryId}`);
    if (!res.ok) {
      throw new Error(`microscope: getEntry failed with status ${res.status}`);
    }
    return res.json();
  }

  /**
   * Periodically record this Node.js process's runtime metrics (event loop
   * utilization, memory) so the dashboard's metrics view has something to
   * show for Node services, the same way it does for Go. No-op in a browser.
   * Safe to call once at startup; a second call is a no-op unless
   * `stopRuntimeMetrics` was called first.
   */
  async startRuntimeMetrics(intervalMs = 15_000): Promise<void> {
    if (this.metricsTimer !== null) return;
    const { isNodeRuntime, sampleRuntimeMetrics } = await import("./runtimeMetrics.js");
    if (!isNodeRuntime()) return;

    this.metricsTimer = setInterval(() => {
      void this.record("node.runtime", sampleRuntimeMetrics()).catch(() => {});
    }, intervalMs);
    this.metricsTimer.unref?.();
  }

  stopRuntimeMetrics(): void {
    if (this.metricsTimer !== null) {
      clearInterval(this.metricsTimer);
      this.metricsTimer = null;
    }
  }
}
