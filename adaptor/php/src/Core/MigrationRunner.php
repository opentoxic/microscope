<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core;

use Opentoxic\Microscope\Adaptor\Php\Core\Store\StoreInterface;
use PDO;

final class MigrationRunner
{
    private const TABLE = 'microscope_schema_migrations';

    /** @var list<string> */
    private const FILES = [
        '001_microscope.up.sql',
        '002_microscope_settings.up.sql',
        '003_microscope_options.up.sql',
    ];

    public function __construct(
        private readonly PDO $pdo,
        private readonly string $migrationsPath,
    ) {
    }

    public function up(): void
    {
        $this->pdo->exec(sprintf(
            'CREATE TABLE IF NOT EXISTS %s (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())',
            self::TABLE
        ));

        foreach (self::FILES as $file) {
            $version = str_replace('.up.sql', '', $file);
            $check = $this->pdo->prepare(sprintf('SELECT 1 FROM %s WHERE version = :version', self::TABLE));
            $check->execute(['version' => $version]);
            if ($check->fetchColumn()) {
                continue;
            }

            $sql = file_get_contents(rtrim($this->migrationsPath, '/\\') . DIRECTORY_SEPARATOR . $file);
            if ($sql === false) {
                throw new \RuntimeException("Migration file missing: {$file}");
            }
            $this->pdo->exec($sql);
            $insert = $this->pdo->prepare(sprintf('INSERT INTO %s (version) VALUES (:version)', self::TABLE));
            $insert->execute(['version' => $version]);
        }
    }
}
