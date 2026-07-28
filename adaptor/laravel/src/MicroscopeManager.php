<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Laravel;

use Illuminate\Support\Facades\DB;
use Opentoxic\Microscope\Adaptor\Php\Microscope;
use Opentoxic\Microscope\Adaptor\Php\Setup;

final class MicroscopeManager
{
    private ?Microscope $microscope = null;

    public function instance(): ?Microscope
    {
        if ($this->microscope !== null) {
            return $this->microscope;
        }

        if (! config('microscope.enabled')) {
            return null;
        }

        $connection = DB::connection();
        $config = $connection->getConfig();
        $dsn = sprintf(
            'pgsql:host=%s;port=%s;dbname=%s',
            $config['host'] ?? '127.0.0.1',
            $config['port'] ?? 5432,
            $config['database'] ?? '',
        );
        if (! empty($config['username'])) {
            $dsn .= ';user=' . $config['username'];
        }
        if (! empty($config['password'])) {
            $dsn .= ';password=' . $config['password'];
        }

        $this->microscope = Setup::boot(
            $dsn,
            (string) config('app.env', 'production'),
        );

        return $this->microscope;
    }
}
