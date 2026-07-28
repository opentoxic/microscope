<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php;

final class RuntimeMetrics
{
    /** @return array{name: string, language: string, value: int, unit: string, memory_mb: float, included_files: int} */
    public static function sample(): array
    {
        return [
            'name' => 'php.runtime',
            'language' => 'php',
            'value' => 1,
            'unit' => 'process',
            'memory_mb' => round(memory_get_usage(true) / 1048576, 2),
            'included_files' => count(get_included_files()),
        ];
    }
}
