<?php

declare(strict_types=1);

namespace Qobly\Microscope;

class MicroscopeClient
{
    private string $baseUrl;
    private int $timeoutSeconds;

    public function __construct(string $baseUrl, int $timeoutSeconds = 5)
    {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->timeoutSeconds = $timeoutSeconds;
    }

    public function record(string $name, array $content = []): string
    {
        $body = $this->request('POST', '/api/entries', ['name' => $name, 'content' => $content]);

        return $body['id'];
    }

    public function listEntries(?string $type = null, ?string $search = null, ?int $limit = null, ?int $offset = null): array
    {
        $query = array_filter([
            'type' => $type,
            'search' => $search,
            'limit' => $limit,
            'offset' => $offset,
        ], static fn ($value) => $value !== null);

        $path = '/api/entries' . ($query ? '?' . http_build_query($query) : '');

        return $this->request('GET', $path);
    }

    public function getEntry(string $entryId): array
    {
        return $this->request('GET', '/api/entries/' . rawurlencode($entryId));
    }

    /**
     * Records this process's runtime metrics (memory, included files) once.
     * PHP-FPM handles one request per process, so there is no background
     * timer here — call this per-request (e.g. from middleware) instead.
     */
    public function recordRuntimeMetrics(): string
    {
        $metrics = RuntimeMetrics::sample();

        return $this->record($metrics['name'], $metrics);
    }

    /** @codeCoverageIgnore overridden by tests to avoid real HTTP calls */
    protected function request(string $method, string $path, ?array $body = null): array
    {
        $ch = curl_init($this->baseUrl . $path);
        $headers = ['Accept: application/json'];

        curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $method);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, $this->timeoutSeconds);

        if ($body !== null) {
            $headers[] = 'Content-Type: application/json';
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($body, JSON_THROW_ON_ERROR));
        }

        curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);

        $response = curl_exec($ch);
        if ($response === false) {
            $error = curl_error($ch);
            curl_close($ch);
            throw new \RuntimeException("microscope: request failed: {$error}");
        }

        $status = curl_getinfo($ch, CURLINFO_RESPONSE_CODE);
        curl_close($ch);

        if ($status < 200 || $status >= 300) {
            throw new \RuntimeException("microscope: request failed with status {$status}");
        }

        return json_decode($response, true, flags: JSON_THROW_ON_ERROR);
    }
}
