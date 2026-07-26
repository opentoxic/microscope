# microscope_client (Ruby)

Thin HTTP client for the [microscope](https://github.com/qobly/microscope) observability API.
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
