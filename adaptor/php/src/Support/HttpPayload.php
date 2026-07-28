<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Support;

use Opentoxic\Microscope\Adaptor\Php\Core\Hub;

final class HttpPayload
{
    /**
     * @param  array<string, list<string>>  $headers
     * @return array<string, mixed>
     */
    public static function buildRequestContent(
        Hub $hub,
        string $method,
        string $path,
        string $query,
        int $status,
        float $durationMs,
        string $ip,
        string $userAgent,
        array $headers,
        string $requestBody,
        string $responseBody,
    ): array {
        $redact = $hub->redactSensitive();

        return [
            'method' => $method,
            'path' => $path,
            'query' => $query,
            'status' => $status,
            'duration_ms' => round($durationMs, 2),
            'ip' => $ip,
            'user_agent' => $userAgent,
            'headers' => Sanitizer::headers($headers, $redact),
            'request_body' => Sanitizer::jsonBody($requestBody, $redact),
            'response_body' => Sanitizer::jsonBody($responseBody, $redact),
        ];
    }

    public static function truncate(string $body, int $maxBytes): string
    {
        if ($maxBytes <= 0 || strlen($body) <= $maxBytes) {
            return $body;
        }

        return substr($body, 0, $maxBytes);
    }

    public static function shouldSkipPath(string $path, string $microscopePath): bool
    {
        $prefix = trim($microscopePath, '/');

        return $prefix !== '' && str_starts_with(trim($path, '/'), $prefix);
    }

    public static function isMicroscopeSql(string $sql): bool
    {
        return str_contains(strtolower($sql), 'microscope_entries')
            || str_contains(strtolower($sql), 'microscope_settings')
            || str_contains(strtolower($sql), 'microscope_options');
    }
}
