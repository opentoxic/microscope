<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Tests;

use Opentoxic\Microscope\Adaptor\Php\Support\Sanitizer;
use PHPUnit\Framework\TestCase;

final class SanitizerTest extends TestCase
{
    public function testMapRedactsSensitiveKeys(): void
    {
        $data = [
            'username' => 'alice',
            'password' => 'secret',
            'profile' => ['token' => 'abc123'],
        ];

        $redacted = Sanitizer::map($data, true);

        $this->assertSame('alice', $redacted['username']);
        $this->assertSame('[REDACTED]', $redacted['password']);
        $this->assertSame('[REDACTED]', $redacted['profile']['token']);
    }

    public function testMapLeavesDataUntouchedWhenRedactionDisabled(): void
    {
        $data = ['password' => 'secret'];

        $this->assertSame($data, Sanitizer::map($data, false));
    }

    public function testHeadersRedactsSensitiveValues(): void
    {
        $headers = [
            'Content-Type' => ['application/json'],
            'Authorization' => ['Bearer token'],
        ];

        $redacted = Sanitizer::headers($headers, true);

        $this->assertSame(['application/json'], $redacted['Content-Type']);
        $this->assertSame(['[REDACTED]'], $redacted['Authorization']);
    }
}
