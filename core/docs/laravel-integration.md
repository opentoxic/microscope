# Laravel integration

Packages:

- `opentoxic/microscope-adaptor-laravel` — provider, middleware, routes
- `opentoxic/microscope-adaptor-php` — native hub, store, API (required dependency)

Laravel runs microscope **in-process**. No Go sidecar or HTTP proxy.

## Install

### Packagist (when published)

```bash
composer require opentoxic/microscope-adaptor-laravel
php artisan vendor:publish --tag=microscope-config
php artisan migrate
```

### Monorepo / path repository

```json
{
    "repositories": [
        { "type": "path", "url": "../microscope/adaptor/laravel" },
        { "type": "path", "url": "../microscope/adaptor/php" }
    ],
    "require": {
        "opentoxic/microscope-adaptor-laravel": "dev-main",
        "opentoxic/microscope-adaptor-php": "dev-main"
    },
    "minimum-stability": "dev",
    "prefer-stable": true
}
```

Both path repos are required — Composer does not read nested `repositories` from the Laravel package.

```bash
composer update opentoxic/microscope-adaptor-laravel opentoxic/microscope-adaptor-php
```

## Configuration

`.env`:

```env
APP_ENV=local
MICROSCOPE_ENABLED=true
MICROSCOPE_PATH=microscope
MICROSCOPE_MAX_BODY_BYTES=65536
MICROSCOPE_REDACT_SENSITIVE=false
```

Published `config/microscope.php` defaults `enabled` to `false` — you must set `MICROSCOPE_ENABLED=true`.

## Register the provider

Laravel 11+ auto-discovers the package provider. Otherwise add to `bootstrap/providers.php`:

```php
Opentoxic\Microscope\Adaptor\Laravel\MicroscopeServiceProvider::class,
```

## Request capture middleware

Register `RecordRequests` globally or on the `web` group in `bootstrap/app.php`:

```php
use Opentoxic\Microscope\Adaptor\Laravel\Middleware\RecordRequests;

->withMiddleware(function (Middleware $middleware) {
    $middleware->append(RecordRequests::class);
})
```

This captures full HTTP request/response bodies, headers, query string, and links SQL queries to the request batch.

## Migrations

Package migrations ship with the Laravel adaptor and load via `loadMigrationsFrom()`:

```bash
php artisan migrate
```

Creates `microscope_entries`, `microscope_settings`, `microscope_options` (idempotent checks included).

Tables may also be created on first dashboard visit when `MICROSCOPE_AUTO_MIGRATE=true`.

## Runtime metrics

PHP runtime metrics (`php.runtime`) are recorded:

- On incoming HTTP requests (throttled to every 15 seconds)
- Via scheduled task every minute if `schedule:work` or cron is running

Make normal requests to your app, then refresh the dashboard — the **Service interactions** center should show **PHP**.

## Auth in non-local environments

By default the dashboard uses the `web` middleware stack. For staging, add auth:

```php
// config/microscope.php
'middleware' => ['web', 'auth', 'can:viewMicroscope'],
```

Define a `viewMicroscope` gate in your `AppServiceProvider`.

## Verify

1. `php artisan serve`
2. Visit a few application routes
3. Open `http://127.0.0.1:8000/microscope`
4. Open a **request** entry — Payload, Headers, and Response tabs should have data (new requests only)

## Artisan commands

```bash
php artisan microscope:install   # publish config
```

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| 404 on `/microscope` | Check `MICROSCOPE_ENABLED`, `APP_ENV` in allowed envs, provider registered |
| Empty request bodies | Entries recorded before middleware upgrade have no payloads — make new requests |
| Center shows "GO" | No `php.runtime` metrics yet — hit app routes, wait 15s, refresh |
| `ext-pcntl` on Windows | Add platform fake in root `composer.json` if other packages require it |

## See also

- [Configuration](configuration.md)
- [Architecture](architecture.md) — PHP core vs Laravel adaptor
- [Custom events](tutorials/custom-events.md)
