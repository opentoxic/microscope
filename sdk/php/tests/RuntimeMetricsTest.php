<?php

declare(strict_types=1);

namespace Qobly\Microscope\Tests;

use PHPUnit\Framework\TestCase;
use Qobly\Microscope\RuntimeMetrics;

final class RuntimeMetricsTest extends TestCase
{
    public function testSampleReturnsExpectedShape(): void
    {
        $metrics = RuntimeMetrics::sample();

        self::assertSame('php.runtime', $metrics['name']);
        self::assertSame('php', $metrics['language']);
        self::assertSame('MB', $metrics['unit']);
        self::assertIsFloat($metrics['value']);
        self::assertGreaterThan(0, $metrics['value']);
        self::assertIsFloat($metrics['memory_mb']);
        self::assertIsFloat($metrics['peak_memory_mb']);
        self::assertIsInt($metrics['included_files']);
        self::assertGreaterThan(0, $metrics['included_files']);
    }
}
