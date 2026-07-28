<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Laravel\Middleware;

use Closure;
use Illuminate\Http\Request;
use Opentoxic\Microscope\Adaptor\Laravel\MicroscopeManager;
use Opentoxic\Microscope\Adaptor\Laravel\Support\RequestContext;
use Opentoxic\Microscope\Adaptor\Php\Core\EntryType;
use Opentoxic\Microscope\Adaptor\Php\Support\HttpPayload;
use Symfony\Component\HttpFoundation\Response;

final class RecordRequests
{
    public function __construct(private readonly MicroscopeManager $manager)
    {
    }

    public function handle(Request $request, Closure $next): Response
    {
        if (! config('microscope.enabled')) {
            return $next($request);
        }

        $microscopePath = (string) config('microscope.path', 'microscope');
        if (HttpPayload::shouldSkipPath($request->path(), $microscopePath)) {
            return $next($request);
        }

        $batchId = RequestContext::start($request);
        $started = microtime(true);
        $maxBodyBytes = (int) config('microscope.max_body_bytes', 65536);

        $hub = $this->manager->instance()?->hub();
        $hub?->recordRuntimeMetricsIfDue();

        $requestBody = HttpPayload::truncate($request->getContent(), $maxBodyBytes);

        /** @var Response $response */
        $response = $next($request);

        $microscope = $this->manager->instance();
        $hub = $microscope?->hub();
        if ($hub === null) {
            return $response;
        }

        $responseBody = '';
        if (! $response->isRedirection() && $response->getContent() !== false) {
            $responseBody = HttpPayload::truncate((string) $response->getContent(), $maxBodyBytes);
        }

        $content = HttpPayload::buildRequestContent(
            hub: $hub,
            method: $request->method(),
            path: '/' . ltrim($request->path(), '/'),
            query: $request->getQueryString() ?? '',
            status: $response->getStatusCode(),
            durationMs: (microtime(true) - $started) * 1000,
            ip: $request->ip() ?? '',
            userAgent: $request->userAgent() ?? '',
            headers: $request->headers->all(),
            requestBody: $requestBody,
            responseBody: $responseBody,
        );

        $requestId = $request->header('X-Request-Id') ?? $request->header('X-Correlation-Id');

        $hub->record(
            EntryType::REQUEST,
            $content,
            $batchId,
            is_string($requestId) ? $requestId : null,
            $batchId,
        );

        return $response;
    }
}
