# opentoxic/microscope-client (PHP)

Thin HTTP client for the [microscope](https://github.com/opentoxic/microscope) observability API.
Only depends on ext-curl and ext-json — no third-party HTTP library required.

## Install

```bash
composer require opentoxic/microscope-client
```

## Usage

```php
use Opentoxic\Microscope\MicroscopeClient;

$client = new MicroscopeClient('http://localhost:8093/microscope');

$client->record('payment_charged', ['amount' => 4200]);
$entries = $client->listEntries(type: 'custom', limit: 20);
$entry = $client->getEntry($entries['items'][0]['id']);
```

## Runtime metrics

PHP-FPM handles one request per process, so there's no background timer here — call this
once per request (or on a schedule) instead:

```php
$client->recordRuntimeMetrics(); // memory, peak memory, included files
```

## Testing

```bash
composer install
composer test
```

## Laravel integration

This package ships a service provider that wires the client into the container and records
every request automatically. Laravel's package auto-discovery picks it up once installed —
see [`src/Laravel/README.md`](src/Laravel/README.md) for configuration.
