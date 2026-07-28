<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core;

final class EntryDetailBuilder
{
    /** @param list<Entry> $batch */
    public static function build(Entry $entry, array $batch): array
    {
        $batchArrays = array_map(static fn (Entry $e) => $e->toArray(), $batch);
        $groups = self::groupBatchByType(array_values(array_filter(
            $batch,
            static fn (Entry $e) => $e->id !== $entry->id,
        )));

        return [
            'entry' => $entry->toArray(),
            'batch' => $batchArrays,
            'batch_groups' => $groups,
            'content_tabs' => self::contentTabs($entry),
            'related_active_tab' => $groups[0]['type'] ?? '',
        ];
    }

    /**
     * @param  list<Entry>  $batch
     * @return list<array{type: string, label: string, entries: list<array<string, mixed>>}>
     */
    private static function groupBatchByType(array $batch): array
    {
        $byType = [];
        foreach ($batch as $entry) {
            $byType[$entry->type][] = $entry->toArray();
        }

        $groups = [];
        foreach (EntryType::ALL as $type) {
            if (empty($byType[$type])) {
                continue;
            }
            $groups[] = [
                'type' => $type,
                'label' => self::typeLabel($type),
                'entries' => $byType[$type],
            ];
        }

        return $groups;
    }

    /** @return list<array{id: string, label: string, body: string, json: bool}> */
    private static function contentTabs(Entry $entry): array
    {
        $content = $entry->content;
        $tabs = [];

        switch ($entry->type) {
            case EntryType::REQUEST:
                $requestBody = self::contentString($content['request_body'] ?? null);
                if ($requestBody !== '' && $requestBody !== 'null' && $requestBody !== '{}') {
                    $tabs[] = [
                        'id' => 'payload',
                        'label' => 'Payload',
                        'body' => self::prettyContent($requestBody),
                        'json' => self::looksLikeJson($requestBody),
                    ];
                }
                if (! empty($content['headers']) && is_array($content['headers'])) {
                    $tabs[] = [
                        'id' => 'headers',
                        'label' => 'Headers',
                        'body' => self::jsonPretty($content['headers']),
                        'json' => true,
                    ];
                }
                $responseBody = self::contentString($content['response_body'] ?? null);
                if ($responseBody !== '' && $responseBody !== 'null') {
                    $tabs[] = [
                        'id' => 'response',
                        'label' => 'Response',
                        'body' => self::prettyContent($responseBody),
                        'json' => self::looksLikeJson($responseBody),
                    ];
                }
                break;

            case EntryType::QUERY:
                $sql = self::contentString($content['sql'] ?? null);
                if ($sql !== '') {
                    $tabs[] = ['id' => 'query', 'label' => 'Query', 'body' => $sql, 'json' => false];
                }
                if (! empty($content['args'])) {
                    $tabs[] = [
                        'id' => 'bindings',
                        'label' => 'Bindings',
                        'body' => self::jsonPretty($content['args']),
                        'json' => true,
                    ];
                }
                break;

            case EntryType::EXCEPTION:
                $stack = self::contentString($content['stack'] ?? null);
                if ($stack !== '') {
                    $tabs[] = ['id' => 'stack', 'label' => 'Stack Trace', 'body' => $stack, 'json' => false];
                }
                break;

            default:
                break;
        }

        if ($tabs === []) {
            $tabs[] = [
                'id' => 'payload',
                'label' => 'Payload',
                'body' => self::jsonPretty($content),
                'json' => true,
            ];
        }

        return $tabs;
    }

    private static function typeLabel(string $type): string
    {
        return match ($type) {
            EntryType::REQUEST => 'Requests',
            EntryType::QUERY => 'Queries',
            EntryType::LOG => 'Logs',
            EntryType::EVENT => 'Events',
            EntryType::NOTIFICATION => 'Notifications',
            EntryType::EXCEPTION => 'Exceptions',
            EntryType::CACHE => 'Cache',
            EntryType::REDIS => 'Redis',
            EntryType::JOB => 'Queue Jobs',
            EntryType::SCHEDULE => 'Scheduled Tasks',
            EntryType::MAIL => 'Mail',
            EntryType::HTTP_CLIENT => 'External Calls',
            EntryType::WEBSOCKET => 'WebSockets',
            EntryType::PERFORMANCE => 'Performance',
            EntryType::METRIC => 'Metrics',
            EntryType::CUSTOM => 'Custom Events',
            EntryType::TOPIC => 'Redpanda Topics',
            default => $type,
        };
    }

    private static function contentString(mixed $value): string
    {
        if ($value === null) {
            return '';
        }
        if (is_string($value)) {
            return $value;
        }
        if (is_scalar($value)) {
            return (string) $value;
        }

        return self::jsonPretty($value);
    }

    private static function prettyContent(string $value): string
    {
        return self::looksLikeJson($value) ? self::prettyJsonString($value) : $value;
    }

    private static function looksLikeJson(string $value): bool
    {
        $trimmed = trim($value);

        return $trimmed !== '' && (str_starts_with($trimmed, '{') || str_starts_with($trimmed, '['))
            && json_validate($trimmed);
    }

    private static function prettyJsonString(string $value): string
    {
        $decoded = json_decode(trim($value), true);
        if (! is_array($decoded)) {
            return $value;
        }

        return json_encode($decoded, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?: $value;
    }

    private static function jsonPretty(mixed $value): string
    {
        return json_encode($value, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?: '{}';
    }
}
