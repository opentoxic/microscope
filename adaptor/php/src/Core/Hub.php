<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core;

use Opentoxic\Microscope\Adaptor\Php\Core\Store\StoreInterface;
use Opentoxic\Microscope\Adaptor\Php\RuntimeMetrics;

final class Hub
{
    /** @var array<string, bool> */
    private array $enabled;

    private bool $recordingPaused = false;

    private bool $redactSensitive;

    /** @var list<callable(Entry): void> */
    private array $subscribers = [];

    /** @var list<callable(array<string, mixed>): void> */
    private array $controlSubscribers = [];

    private ?float $lastRuntimeMetricAt = null;

    private const RUNTIME_METRIC_INTERVAL_SECONDS = 15;

    public function __construct(
        private readonly StoreInterface $store,
        private readonly Config $config,
    ) {
        $this->enabled = array_fill_keys(EntryType::ALL, true);
        $this->redactSensitive = $config->redactSensitive;
        $this->loadTypeSettings();
        $this->loadRedactionSetting();
    }

    public function config(): Config
    {
        return $this->config;
    }

    public function store(): StoreInterface
    {
        return $this->store;
    }

    /** @param array<string, mixed> $content */
    public function record(string $type, array $content, ?string $batchId = null, ?string $requestId = null, ?string $id = null): string
    {
        if ($this->recordingPaused || ! $this->typeEnabled($type)) {
            return '';
        }

        $entryId = $id ?? $this->newId();
        $entry = new Entry(
            id: $entryId,
            batchId: $batchId ?? $entryId,
            type: $type,
            content: $content,
            createdAt: gmdate('Y-m-d\TH:i:s\Z'),
            requestId: $requestId ?? '',
        );

        $this->store->insert($entry);
        $this->publish($entry);

        return $entryId;
    }

    public function recordEntry(Entry $entry): void
    {
        if ($this->recordingPaused || ! $this->typeEnabled($entry->type)) {
            return;
        }
        if ($entry->id === '') {
            $entry->id = $this->newId();
        }
        if ($entry->batchId === '') {
            $entry->batchId = $entry->id;
        }
        $this->store->insert($entry);
        $this->publish($entry);
    }

    public function recordRuntimeMetricsIfDue(): void
    {
        $now = microtime(true);
        if (
            $this->lastRuntimeMetricAt !== null
            && ($now - $this->lastRuntimeMetricAt) < self::RUNTIME_METRIC_INTERVAL_SECONDS
        ) {
            return;
        }

        $this->lastRuntimeMetricAt = $now;

        if ($this->recordingPaused || ! $this->typeEnabled(EntryType::METRIC)) {
            return;
        }

        $this->record(EntryType::METRIC, RuntimeMetrics::sample());
    }

    public function typeEnabled(string $type): bool
    {
        return EntryType::isValid($type) && ($this->enabled[$type] ?? false);
    }

    public function recordingPaused(): bool
    {
        return $this->recordingPaused;
    }

    public function setRecordingPaused(bool $paused): void
    {
        $this->recordingPaused = $paused;
        $this->publishControl(['action' => 'recording-paused', 'paused' => $paused]);
    }

    public function redactSensitive(): bool
    {
        return $this->redactSensitive;
    }

    public function setRedactSensitive(bool $enabled): void
    {
        $this->redactSensitive = $enabled;
        $this->store->setOption('redact_sensitive', json_encode($enabled, JSON_THROW_ON_ERROR));
        $this->publishControl(['action' => 'redaction', 'redact_sensitive' => $enabled]);
    }

    /** @return list<array{type: string, enabled: bool, count: int}> */
    public function typeSettings(): array
    {
        return $this->store->listTypeSettings();
    }

    public function setTypeEnabled(string $type, bool $enabled): int
    {
        if (! EntryType::isValid($type)) {
            throw new \InvalidArgumentException("unknown signal type: {$type}");
        }
        $this->enabled[$type] = $enabled;
        $deleted = $this->store->setTypeEnabled($type, $enabled);
        $this->publishControl(['action' => 'signal-setting', 'type' => $type, 'deleted' => $deleted]);

        return $deleted;
    }

    public function subscribe(callable $callback): void
    {
        $this->subscribers[] = $callback;
    }

    public function subscribeControl(callable $callback): void
    {
        $this->controlSubscribers[] = $callback;
    }

    private function publish(Entry $entry): void
    {
        foreach ($this->subscribers as $callback) {
            $callback($entry);
        }
    }

    /** @param array<string, mixed> $event */
    private function publishControl(array $event): void
    {
        foreach ($this->controlSubscribers as $callback) {
            $callback($event);
        }
    }

    private function loadTypeSettings(): void
    {
        foreach ($this->store->listTypeSettings() as $setting) {
            $this->enabled[$setting['type']] = $setting['enabled'];
        }
    }

    private function loadRedactionSetting(): void
    {
        $raw = $this->store->getOption('redact_sensitive');
        if ($raw !== null) {
            $decoded = json_decode($raw, true);
            if (is_bool($decoded)) {
                $this->redactSensitive = $decoded;
            }
        }
    }

    private function newId(): string
    {
        return bin2hex(random_bytes(16));
    }
}
