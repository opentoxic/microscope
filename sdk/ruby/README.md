# microscope_client (Ruby)

Thin HTTP client for the [microscope](https://github.com/opentoxic/microscope) observability API.
No dependencies beyond Ruby's standard library.

## Install

```ruby
# Gemfile
gem "microscope_client"
```

## Usage

```ruby
require "microscope_client"

client = MicroscopeClient::Client.new(base_url: "http://localhost:8093/microscope")

client.record("payment_charged", content: { amount: 4200 })
entries = client.list_entries(type: "custom", limit: 20)
entry = client.get_entry(entries["items"].first["id"])
```

## Runtime metrics

Periodically records thread count and GC stats so the dashboard's metrics view has something
to show for Ruby services, the same way it does for Go:

```ruby
client.start_runtime_metrics(interval: 15) # call once at startup
```

## Testing

```bash
rake test
# or
ruby -Ilib -Itest test/client_test.rb test/runtime_metrics_test.rb
```
