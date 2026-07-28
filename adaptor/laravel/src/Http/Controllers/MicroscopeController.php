<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Laravel\Http\Controllers;

use Illuminate\Http\Request;
use Opentoxic\Microscope\Adaptor\Laravel\MicroscopeManager;
use Symfony\Component\HttpFoundation\Response;
use Symfony\Component\HttpFoundation\StreamedResponse;

final class MicroscopeController
{
    public function __construct(private readonly MicroscopeManager $manager)
    {
    }

    public function __invoke(Request $request, ?string $path = null): Response
    {
        $microscope = $this->manager->instance();
        if ($microscope === null || ! $microscope->active) {
            abort(404);
        }

        $prefix = '/' . trim((string) config('microscope.path', 'microscope'), '/');
        $targetPath = $prefix . ($path !== null && $path !== '' ? '/' . $path : '');

        if (str_contains($targetPath, '/api/stream')) {
            return response()->stream(function () use ($microscope): void {
                $microscope->stream(function (string $chunk): void {
                    echo $chunk;
                    if (ob_get_level() > 0) {
                        ob_flush();
                    }
                    flush();
                });
            }, 200, [
                'Content-Type' => 'text/event-stream',
                'Cache-Control' => 'no-cache',
                'X-Accel-Buffering' => 'no',
            ]);
        }

        $result = $microscope->handle(
            $request->getMethod(),
            $targetPath,
            $request->getContent(),
        );

        if ($result === null) {
            abort(404);
        }

        return response($result['body'], $result['status'], $result['headers'] ?? []);
    }
}
