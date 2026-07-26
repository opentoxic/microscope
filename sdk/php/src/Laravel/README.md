# Laravel integration

Included in `qobly/microscope-client` — no separate package needed. Laravel's package
auto-discovery registers `MicroscopeServiceProvider` automatically once the package is required.

## Setup

```bash
composer require qobly/microscope-client
```

```
# .env
MICROSCOPE_BASE_URL=http://localhost:8093/microscope
```

Optionally publish the config file:

```bash
php artisan vendor:publish --tag=microscope-config
```

## Record requests automatically

Register the middleware in `bootstrap/app.php` (Laravel 11+) or `App\Http\Kernel` (Laravel 10):

```php
$middleware->append(\Qobly\Microscope\Laravel\RecordRequests::class);
```

## Record custom entries

Resolve `MicroscopeClient` from the container anywhere:

```php
use Qobly\Microscope\MicroscopeClient;

app(MicroscopeClient::class)->record('payment_charged', ['amount' => 4200]);
```

## Runtime metrics

PHP-FPM runs one request per process, so there's no background timer like the other SDKs have.
Record a sample from a scheduled command instead, so the dashboard's metrics view has
something to show for PHP services:

```php
// app/Console/Kernel.php
$schedule->call(fn () => app(MicroscopeClient::class)->recordRuntimeMetrics())->everyMinute();
```
