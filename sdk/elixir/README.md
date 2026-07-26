# microscope_client (Elixir)

Thin HTTP client for the [microscope](https://github.com/qobly/microscope) observability API.

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
