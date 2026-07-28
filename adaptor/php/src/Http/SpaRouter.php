<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Http;

final class SpaRouter
{
    public function __construct(private readonly string $distPath, private readonly string $pathPrefix)
    {
    }

    /** @return array{status: int, headers: array<string, string>, body: string}|null */
    public function handle(string $method, string $path): ?array
    {
        if ($method !== 'GET') {
            return null;
        }

        $prefix = rtrim($this->pathPrefix, '/');
        if (! str_starts_with($path, $prefix)) {
            return null;
        }

        $relative = ltrim(substr($path, strlen($prefix)), '/');
        if (str_starts_with($relative, 'api/')) {
            return null;
        }

        if ($relative === '' || $relative === 'settings' || str_starts_with($relative, 'entries/')) {
            return $this->serveFile('index.html', 'text/html; charset=utf-8');
        }

        if (str_starts_with($relative, 'assets/')) {
            return $this->serveFile($relative, $this->mimeType($relative));
        }

        return null;
    }

    /** @return array{status: int, headers: array<string, string>, body: string} */
    private function serveFile(string $relative, string $contentType): array
    {
        $file = rtrim($this->distPath, '/\\') . DIRECTORY_SEPARATOR . str_replace('/', DIRECTORY_SEPARATOR, $relative);
        if (! is_file($file)) {
            return ['status' => 404, 'headers' => ['Content-Type' => 'text/plain'], 'body' => 'not found'];
        }

        return [
            'status' => 200,
            'headers' => ['Content-Type' => $contentType],
            'body' => (string) file_get_contents($file),
        ];
    }

    private function mimeType(string $path): string
    {
        return match (pathinfo($path, PATHINFO_EXTENSION)) {
            'js' => 'application/javascript',
            'css' => 'text/css',
            'svg' => 'image/svg+xml',
            'png' => 'image/png',
            'woff2' => 'font/woff2',
            default => 'application/octet-stream',
        };
    }
}
