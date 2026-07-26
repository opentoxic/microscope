# qobly/microscope-client (PHP)

Thin HTTP client for the [microscope](https://github.com/qobly/microscope) observability API.
Only depends on ext-curl and ext-json — no third-party HTTP library required.

## Install

```bash
composer require qobly/microscope-client
```

## Usage

```php
use Qobly\Microscope\MicroscopeClient;

$client = new MicroscopeClient('http://localhost:8093/microscope');

$client->record('payment_charged', ['amount' => 4200]);
$entries = $client->listEntries(type: 'custom', limit: 20);
$entry = $client->getEntry($entries['items'][0]['id']);
```

## Laravel integration

This package ships a service provider that wires the client into the container and records
every request automatically. Laravel's package auto-discovery picks it up once installed —
see [`src/Laravel/README.md`](src/Laravel/README.md) for configuration.
