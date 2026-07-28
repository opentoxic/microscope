<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Laravel;

use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\ServiceProvider;
use Opentoxic\Microscope\Adaptor\Laravel\Commands\InstallCommand;
use Opentoxic\Microscope\Adaptor\Laravel\Http\Controllers\MicroscopeController;
use Opentoxic\Microscope\Adaptor\Laravel\Support\RequestContext;
use Opentoxic\Microscope\Adaptor\Php\Core\EntryType;
use Opentoxic\Microscope\Adaptor\Php\Support\HttpPayload;
use Opentoxic\Microscope\Adaptor\Php\Support\Sanitizer;

final class MicroscopeServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__ . '/../config/microscope.php', 'microscope');
        $this->app->singleton(MicroscopeManager::class);
    }

    public function boot(): void
    {
        $this->loadMigrationsFrom(__DIR__ . '/../database/migrations');
        $this->registerRoutes();
        $this->registerCollectors();

        $this->publishes([
            __DIR__ . '/../config/microscope.php' => config_path('microscope.php'),
        ], 'microscope-config');

        if ($this->app->runningInConsole()) {
            $this->commands([InstallCommand::class]);
        }
    }

    private function registerRoutes(): void
    {
        if (! config('microscope.enabled')) {
            return;
        }

        $path = trim((string) config('microscope.path', 'microscope'), '/');

        Route::middleware(config('microscope.middleware', ['web']))
            ->prefix($path)
            ->group(function (): void {
                Route::any('/{path?}', MicroscopeController::class)
                    ->where('path', '.*')
                    ->name('microscope');
            });
    }

    private function registerCollectors(): void
    {
        if (! config('microscope.enabled')) {
            return;
        }

        DB::listen(function ($query) {
            if (HttpPayload::isMicroscopeSql($query->sql)) {
                return;
            }

            $manager = $this->app->make(MicroscopeManager::class);
            $hub = $manager->instance()?->hub();
            if ($hub === null) {
                return;
            }

            $batchId = RequestContext::batchId();
            $bindings = Sanitizer::bindings($query->bindings ?? [], $hub->redactSensitive());

            $hub->record(EntryType::QUERY, [
                'sql' => $query->sql,
                'args' => $bindings,
                'duration_ms' => $query->time,
                'connection' => $query->connectionName,
            ], $batchId);
        });
    }
}
