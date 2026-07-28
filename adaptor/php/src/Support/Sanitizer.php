<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Support;

final class Sanitizer
{
    /** @var list<string> */
    private const SENSITIVE_KEYS = [
        'password', 'password_hash', 'new_password', 'old_password', 'current_password',
        'refresh_token', 'access_token', 'token', 'otp', 'code', 'secret',
        'encryption_key', 'authorization', 'mfa_secret', 'backup_codes',
    ];

    /** @var list<string> */
    private const SENSITIVE_HEADERS = [
        'authorization', 'cookie', 'x-api-key', 'x-auth-token',
    ];

    /**
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    public static function map(array $data, bool $redact): array
    {
        if (! $redact) {
            return $data;
        }

        $out = [];
        foreach ($data as $key => $value) {
            $out[$key] = self::value((string) $key, $value, true);
        }

        return $out;
    }

    /**
     * @param  array<string, list<string>>  $headers
     * @return array<string, list<string>>
     */
    public static function headers(array $headers, bool $redact): array
    {
        if (! $redact) {
            return $headers;
        }

        $out = [];
        foreach ($headers as $key => $values) {
            if (in_array(strtolower($key), self::SENSITIVE_HEADERS, true)) {
                $out[$key] = ['[REDACTED]'];
                continue;
            }
            $out[$key] = $values;
        }

        return $out;
    }

    public static function jsonBody(string $body, bool $redact): string
    {
        if ($body === '') {
            return '';
        }

        $decoded = json_decode($body, true);
        if (json_last_error() !== JSON_ERROR_NONE) {
            return $body;
        }

        if ($redact && is_array($decoded)) {
            $decoded = self::map($decoded, true);
        }

        return json_encode($decoded, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?: $body;
    }

    /**
     * @param  list<mixed>  $bindings
     * @return list<mixed>
     */
    public static function bindings(array $bindings, bool $redact): array
    {
        if (! $redact) {
            return $bindings;
        }

        return array_map(static fn ($value) => is_string($value) ? '[REDACTED]' : $value, $bindings);
    }

    private static function value(string $key, mixed $value, bool $redact): mixed
    {
        if ($redact && in_array(strtolower($key), self::SENSITIVE_KEYS, true)) {
            return '[REDACTED]';
        }

        if (is_array($value)) {
            $nested = [];
            foreach ($value as $nestedKey => $nestedValue) {
                $nested[$nestedKey] = self::value((string) $nestedKey, $nestedValue, $redact);
            }

            return $nested;
        }

        return $value;
    }
}
