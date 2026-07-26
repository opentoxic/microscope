<?php

declare(strict_types=1);

namespace Qobly\Microscope\Laravel;

use Closure;
use Illuminate\Http\Request;
use Qobly\Microscope\MicroscopeClient;
use Symfony\Component\HttpFoundation\Response;

final class RecordRequests
{
    public function __construct(private readonly MicroscopeClient $client)
    {
    }

    public function handle(Request $request, Closure $next): Response
    {
        $started = microtime(true);
        $response = $next($request);

        $this->client->record('http_request', [
            'method' => $request->method(),
            'path' => $request->path(),
            'status' => $response->getStatusCode(),
            'duration_ms' => round((microtime(true) - $started) * 1000, 2),
        ]);

        return $response;
    }
}
