# microscope_client (Elixir)

Thin HTTP client for the [microscope](https://github.com/opentoxic/microscope) observability API.

**Docs:** [Custom events](../../core/docs/tutorials/custom-events.md) · [Getting started](../../core/docs/getting-started.md)

## Install

```elixir
def deps do
  [
    {:microscope_client, "~> 0.1"}
  ]
end
```

## Usage

```elixir
client = MicroscopeClient.new("http://localhost:8093/microscope")

{:ok, id} = MicroscopeClient.record(client, "payment_charged", %{amount: 4200})
{:ok, entries} = MicroscopeClient.list_entries(client, type: "custom", limit: 20)
{:ok, entry} = MicroscopeClient.get_entry(client, id)
```

## Runtime metrics

Periodically records process count, scheduler count, and memory so the dashboard's metrics
view has something to show for Elixir services, the same way it does for Go:

```elixir
pid = MicroscopeClient.start_runtime_metrics(client, 15_000) # call once at startup
# later, if needed:
MicroscopeClient.stop_runtime_metrics(pid)
```

## Testing

```bash
mix deps.get
mix test
```
