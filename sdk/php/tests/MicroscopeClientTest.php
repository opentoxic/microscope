<?php

declare(strict_types=1);

namespace Qobly\Microscope\Tests;

use PHPUnit\Framework\TestCase;
use Qobly\Microscope\MicroscopeClient;

/** Overrides the curl transport so tests never hit the network. */
final class FakeMicroscopeClient extends MicroscopeClient
{
    /** @var array<int, array{method: string, path: string, body: ?array}> */
    public array $calls = [];

    /** @var array<string, mixed> */
    public array $nextResponse = ['id' => 'entry-1'];

    protected function request(string $method, string $path, ?array $body = null): array
    {
        $this->calls[] = ['method' => $method, 'path' => $path, 'body' => $body];

        return $this->nextResponse;
    }
}

final class MicroscopeClientTest extends TestCase
{
    private FakeMicroscopeClient $client;

    protected function setUp(): void
    {
        $this->client = new FakeMicroscopeClient('http://localhost:8093/microscope/');
    }

    public function testRecordPostsNameAndContent(): void
    {
        $id = $this->client->record('payment_charged', ['amount' => 4200]);

        self::assertSame('entry-1', $id);
        self::assertCount(1, $this->client->calls);
        self::assertSame('POST', $this->client->calls[0]['method']);
        self::assertSame('/api/entries', $this->client->calls[0]['path']);
        self::assertSame(
            ['name' => 'payment_charged', 'content' => ['amount' => 4200]],
            $this->client->calls[0]['body'],
        );
    }

    public function testListEntriesBuildsQueryString(): void
    {
        $this->client->nextResponse = ['items' => [], 'total' => 0];

        $this->client->listEntries(type: 'custom', limit: 20);

        self::assertSame('GET', $this->client->calls[0]['method']);
        self::assertSame('/api/entries?type=custom&limit=20', $this->client->calls[0]['path']);
    }

    public function testGetEntryBuildsCorrectPath(): void
    {
        $this->client->nextResponse = ['id' => 'entry-1'];

        $entry = $this->client->getEntry('entry-1');

        self::assertSame('/api/entries/entry-1', $this->client->calls[0]['path']);
        self::assertSame(['id' => 'entry-1'], $entry);
    }

    public function testRecordRuntimeMetricsRecordsPhpRuntimeMetrics(): void
    {
        $this->client->recordRuntimeMetrics();

        self::assertSame('/api/entries', $this->client->calls[0]['path']);
        self::assertSame('php.runtime', $this->client->calls[0]['body']['name']);
        self::assertSame('php', $this->client->calls[0]['body']['content']['language']);
    }
}
