<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php;

use Opentoxic\Microscope\Adaptor\Php\Core\Config;
use Opentoxic\Microscope\Adaptor\Php\Core\Hub;
use Opentoxic\Microscope\Adaptor\Php\Core\MigrationRunner;
use Opentoxic\Microscope\Adaptor\Php\Core\Store\PostgresStore;
use Opentoxic\Microscope\Adaptor\Php\Http\ApiRouter;
use Opentoxic\Microscope\Adaptor\Php\Http\SpaRouter;
use PDO;

final class Setup
{
    public static function boot(string $dsn, string $appEnv, ?Config $config = null): Microscope
    {
        $config ??= Config::fromEnv();
        if (! $config->isActive($appEnv)) {
            return new Microscope(null, null, null, false);
        }

        $pdo = new PDO($dsn);
        $pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);

        $migrationsPath = dirname(__DIR__, 3) . DIRECTORY_SEPARATOR . 'core' . DIRECTORY_SEPARATOR . 'migrations';
        if ($config->autoMigrate) {
            (new MigrationRunner($pdo, $migrationsPath))->up();
        }

        $store = new PostgresStore($pdo);
        $hub = new Hub($store, $config);
        $api = new ApiRouter($hub);
        $spa = new SpaRouter(
            dirname(__DIR__) . '/resources/ui/dist',
            $config->pathPrefix(),
        );

        return new Microscope($hub, $api, $spa, true);
    }
}
