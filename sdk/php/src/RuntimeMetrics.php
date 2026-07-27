<?php

declare(strict_types=1);

namespace Opentoxic\Microscope;

/**
 * Best-effort PHP runtime metrics, shaped like every other microscope SDK: a
 * name, a language tag, a primary value + unit, plus language-specific extras.
 *
 * PHP-FPM processes one request per process, so there is no long-running
 * background loop here — call sample() once per request (e.g. from
 * middleware) rather than on a timer.
 */
final class RuntimeMetrics
{
    public static function sample(): array
    {
        $memoryMb = memory_get_usage(true) / 1048576;
        $peakMemoryMb = memory_get_peak_usage(true) / 1048576;

        return [
            'name' => 'php.runtime',
            'language' => 'php',
            'value' => round($memoryMb, 2),
            'unit' => 'MB',
            'memory_mb' => round($memoryMb, 2),
            'peak_memory_mb' => round($peakMemoryMb, 2),
            'included_files' => count(get_included_files()),
        ];
    }
}
