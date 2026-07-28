<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Laravel\Commands;

use Illuminate\Console\Command;

final class InstallCommand extends Command
{
    protected $signature = 'microscope:install';

    protected $description = 'Publish microscope configuration';

    public function handle(): int
    {
        $this->call('vendor:publish', ['--tag' => 'microscope-config']);
        $this->components->info('Microscope config published. Set MICROSCOPE_ENABLED=true in .env');
        $this->components->info('Run: php artisan migrate (creates microscope_entries, microscope_settings, microscope_options on PostgreSQL)');

        return self::SUCCESS;
    }
}
