<?php

declare(strict_types=1);

namespace Qobly\Microscope\Laravel;

use Illuminate\Support\ServiceProvider;
use Qobly\Microscope\MicroscopeClient;

final class MicroscopeServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__ . '/config/microscope.php', 'microscope');

        $this->app->singleton(MicroscopeClient::class, static function ($app) {
            return new MicroscopeClient((string) $app['config']->get('microscope.base_url'));
        });
    }

    public function boot(): void
    {
        $this->publishes([
            __DIR__ . '/config/microscope.php' => config_path('microscope.php'),
        ], 'microscope-config');
    }
}
