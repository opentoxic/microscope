<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core\Store;

use Opentoxic\Microscope\Adaptor\Php\Core\Entry;

interface StoreInterface
{
    public function insert(Entry $entry): void;

    public function get(string $id): ?Entry;

    /** @return array{entries: list<array<string, mixed>>, total: int} */
    public function list(?string $type, ?string $search, int $limit, int $offset): array;

    /** @return list<Entry> */
    public function listByBatch(string $batchId): array;

    public function prune(\DateTimeImmutable $olderThan): int;

    public function clearAll(): int;

    /** @return list<array{type: string, enabled: bool, count: int}> */
    public function listTypeSettings(): array;

    public function setTypeEnabled(string $type, bool $enabled): int;

    public function getOption(string $key): ?string;

    public function setOption(string $key, string $jsonValue): void;

    /** @return array<string, float|int> */
    public function storageUsage(): array;
}
