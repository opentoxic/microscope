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
}
