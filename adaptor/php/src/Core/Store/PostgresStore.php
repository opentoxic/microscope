<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core\Store;

use Opentoxic\Microscope\Adaptor\Php\Core\Entry;
use Opentoxic\Microscope\Adaptor\Php\Core\EntryType;
use PDO;

final class PostgresStore implements StoreInterface
{
    public function __construct(private readonly PDO $pdo)
    {
        $this->pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    }

    public function insert(Entry $entry): void
    {
        $stmt = $this->pdo->prepare(
            'INSERT INTO microscope_entries (id, batch_id, type, request_id, correlation_id, tags, content, created_at)
             VALUES (:id, :batch_id, :type, :request_id, :correlation_id, :tags, :content, :created_at)'
        );
        $stmt->execute([
            'id' => $entry->id,
            'batch_id' => $entry->batchId,
            'type' => $entry->type,
            'request_id' => $entry->requestId !== '' ? $entry->requestId : null,
            'correlation_id' => $entry->correlationId !== '' ? $entry->correlationId : null,
            'tags' => json_encode($entry->tags, JSON_THROW_ON_ERROR),
            'content' => json_encode($entry->content, JSON_THROW_ON_ERROR),
            'created_at' => $entry->createdAt,
        ]);
    }

    public function get(string $id): ?Entry
    {
        $stmt = $this->pdo->prepare(
            'SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at
             FROM microscope_entries WHERE id = :id'
        );
        $stmt->execute(['id' => $id]);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row ? $this->mapRow($row) : null;
    }

    public function list(?string $type, ?string $search, int $limit, int $offset): array
    {
        $limit = max(1, min($limit <= 0 ? 50 : $limit, 200));
        $where = 'WHERE 1=1';
        $params = [];

        if ($type !== null && $type !== '') {
            $where .= ' AND type = :type';
            $params['type'] = $type;
        }
        if ($search !== null && $search !== '') {
            $where .= ' AND (content::text ILIKE :search OR request_id ILIKE :search)';
            $params['search'] = '%' . $search . '%';
        }

        $countStmt = $this->pdo->prepare("SELECT COUNT(*) FROM microscope_entries {$where}");
        $countStmt->execute($params);
        $total = (int) $countStmt->fetchColumn();

        $sql = "SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at
                FROM microscope_entries {$where}
                ORDER BY created_at DESC
                LIMIT :limit OFFSET :offset";
        $stmt = $this->pdo->prepare($sql);
        foreach ($params as $key => $value) {
            $stmt->bindValue(':' . $key, $value);
        }
        $stmt->bindValue(':limit', $limit, PDO::PARAM_INT);
        $stmt->bindValue(':offset', max(0, $offset), PDO::PARAM_INT);
        $stmt->execute();

        $entries = [];
        while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
            $entries[] = $this->mapRow($row)->toArray();
        }

        return ['entries' => $entries, 'total' => $total];
    }

    public function listByBatch(string $batchId): array
    {
        $stmt = $this->pdo->prepare(
            'SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at
             FROM microscope_entries WHERE batch_id = :batch_id ORDER BY created_at ASC'
        );
        $stmt->execute(['batch_id' => $batchId]);
        $entries = [];
        while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
            $entries[] = $this->mapRow($row);
        }

        return $entries;
    }

    public function prune(\DateTimeImmutable $olderThan): int
    {
        $stmt = $this->pdo->prepare('DELETE FROM microscope_entries WHERE created_at < :cutoff');
        $stmt->execute(['cutoff' => $olderThan->format('Y-m-d H:i:sP')]);

        return $stmt->rowCount();
    }

    public function clearAll(): int
    {
        $deleted = $this->pdo->exec('DELETE FROM microscope_entries');
        $this->pdo->exec('VACUUM FULL ANALYZE microscope_entries');

        return (int) $deleted;
    }

    public function listTypeSettings(): array
    {
        $types = EntryType::ALL;
        $placeholders = implode(',', array_fill(0, count($types), '?'));
        $sql = "SELECT configured.type,
                       COALESCE(settings.enabled, TRUE) AS enabled,
                       COALESCE(entries.count, 0) AS count
                FROM unnest(ARRAY[{$placeholders}]::text[]) AS configured(type)
                LEFT JOIN microscope_settings settings ON settings.type = configured.type
                LEFT JOIN (
                    SELECT type, COUNT(*) AS count FROM microscope_entries GROUP BY type
                ) entries ON entries.type = configured.type";
        $stmt = $this->pdo->prepare($sql);
        $stmt->execute($types);
        $settings = [];
        while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
            $settings[] = [
                'type' => $row['type'],
                'enabled' => filter_var($row['enabled'], FILTER_VALIDATE_BOOL),
                'count' => (int) $row['count'],
            ];
        }

        return $settings;
    }

    public function setTypeEnabled(string $type, bool $enabled): int
    {
        $this->pdo->beginTransaction();
        try {
            $stmt = $this->pdo->prepare(
                'INSERT INTO microscope_settings (type, enabled, updated_at)
                 VALUES (:type, :enabled, NOW())
                 ON CONFLICT (type) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at'
            );
            $stmt->execute(['type' => $type, 'enabled' => $enabled ? 'true' : 'false']);

            $deleted = 0;
            if (! $enabled) {
                $del = $this->pdo->prepare('DELETE FROM microscope_entries WHERE type = :type');
                $del->execute(['type' => $type]);
                $deleted = $del->rowCount();
            }
            $this->pdo->commit();

            return $deleted;
        } catch (\Throwable $e) {
            $this->pdo->rollBack();
            throw $e;
        }
    }

    public function getOption(string $key): ?string
    {
        $stmt = $this->pdo->prepare('SELECT value FROM microscope_options WHERE key = :key');
        $stmt->execute(['key' => $key]);
        $value = $stmt->fetchColumn();

        return $value === false ? null : (string) $value;
    }

    public function setOption(string $key, string $jsonValue): void
    {
        $stmt = $this->pdo->prepare(
            'INSERT INTO microscope_options (key, value, updated_at)
             VALUES (:key, :value::jsonb, NOW())
             ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at'
        );
        $stmt->execute(['key' => $key, 'value' => $jsonValue]);
    }

    public function storageUsage(): array
    {
        $sql = "SELECT
            ROUND(pg_total_relation_size('microscope_entries') / 1048576.0, 2) AS entries_mb,
            ROUND(pg_relation_size('microscope_entries') / 1048576.0, 2) AS entries_data_mb,
            ROUND((pg_total_relation_size('microscope_entries') - pg_relation_size('microscope_entries')) / 1048576.0, 2) AS entries_indexes_mb,
            ROUND(pg_total_relation_size('microscope_settings') / 1048576.0, 2) AS settings_mb,
            ROUND(COALESCE((SELECT pg_total_relation_size(oid) FROM pg_class WHERE relname = 'microscope_schema_migrations'), 0) / 1048576.0, 2) AS migrations_mb,
            (SELECT COUNT(*) FROM microscope_entries) AS entry_count";
        $row = $this->pdo->query($sql)->fetch(PDO::FETCH_ASSOC);
        $entriesMb = (float) ($row['entries_mb'] ?? 0);
        $settingsMb = (float) ($row['settings_mb'] ?? 0);
        $migrationsMb = (float) ($row['migrations_mb'] ?? 0);

        return [
            'entries_mb' => $entriesMb,
            'entries_data_mb' => (float) ($row['entries_data_mb'] ?? 0),
            'entries_indexes_mb' => (float) ($row['entries_indexes_mb'] ?? 0),
            'settings_mb' => $settingsMb,
            'migrations_mb' => $migrationsMb,
            'total_mb' => $entriesMb + $settingsMb + $migrationsMb,
            'entry_count' => (int) ($row['entry_count'] ?? 0),
        ];
    }

    /** @param array<string, mixed> $row */
    private function mapRow(array $row): Entry
    {
        $tags = json_decode((string) $row['tags'], true, 512, JSON_THROW_ON_ERROR);
        $content = json_decode((string) $row['content'], true, 512, JSON_THROW_ON_ERROR);

        return new Entry(
            id: (string) $row['id'],
            batchId: (string) $row['batch_id'],
            type: (string) $row['type'],
            content: is_array($content) ? $content : [],
            createdAt: (new \DateTimeImmutable((string) $row['created_at']))->format('Y-m-d\TH:i:s\Z'),
            requestId: (string) ($row['request_id'] ?? ''),
            correlationId: (string) ($row['correlation_id'] ?? ''),
            tags: is_array($tags) ? $tags : [],
        );
    }
}
