# Laravel adaptor

In-process microscope for Laravel — no Go sidecar.

**Documentation:** [core/docs/laravel-integration.md](../../core/docs/laravel-integration.md)

```bash
composer require opentoxic/microscope-adaptor-laravel
php artisan vendor:publish --tag=microscope-config
php artisan migrate
```

Set `MICROSCOPE_ENABLED=true` in `.env` and register `RecordRequests` middleware.
