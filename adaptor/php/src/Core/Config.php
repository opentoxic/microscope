<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core;

final class Config
{
    /** @param list<string> $allowedEnvs */
    public function __construct(
        public readonly bool $enabled = true,
        public readonly string $path = '/microscope',
        public readonly int $retentionHours = 24,
        public readonly int $maxBodyBytes = 65536,
        public readonly array $allowedEnvs = ['development', 'local'],
        public readonly bool $autoMigrate = true,
        public readonly bool $redactSensitive = false,
    ) {
    }

    public static function fromEnv(): self
    {
        $enabled = filter_var(getenv('MICROSCOPE_ENABLED') ?: 'true', FILTER_VALIDATE_BOOL);

        return new self(
            enabled: $enabled,
            path: getenv('MICROSCOPE_PATH') ?: '/microscope',
            retentionHours: (int) (getenv('MICROSCOPE_RETENTION_HOURS') ?: 24),
            maxBodyBytes: (int) (getenv('MICROSCOPE_MAX_BODY_BYTES') ?: 65536),
            allowedEnvs: array_filter(array_map('trim', explode(',', getenv('MICROSCOPE_ALLOWED_ENVS') ?: 'development,local'))),
            autoMigrate: filter_var(getenv('MICROSCOPE_AUTO_MIGRATE') ?: 'true', FILTER_VALIDATE_BOOL),
            redactSensitive: filter_var(getenv('MICROSCOPE_REDACT_SENSITIVE') ?: 'false', FILTER_VALIDATE_BOOL),
        );
    }

    public function pathPrefix(): string
    {
        $path = trim($this->path, '/');

        return $path === '' ? '/microscope' : '/' . $path;
    }

    public function isActive(string $appEnv): bool
    {
        if (! $this->enabled) {
            return false;
        }

        return in_array($appEnv, $this->allowedEnvs, true);
    }
}
