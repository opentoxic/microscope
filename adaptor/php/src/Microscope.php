<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php;

use Opentoxic\Microscope\Adaptor\Php\Core\Hub;
use Opentoxic\Microscope\Adaptor\Php\Http\ApiRouter;
use Opentoxic\Microscope\Adaptor\Php\Http\SpaRouter;

final class Microscope
{
    public function __construct(
        private readonly ?Hub $hub,
        private readonly ?ApiRouter $api,
        private readonly ?SpaRouter $spa,
        public readonly bool $active,
    ) {
    }

    public function hub(): ?Hub
    {
        return $this->hub;
    }

    public function handle(string $method, string $path, string $body = ''): ?array
    {
        if (! $this->active || $this->api === null || $this->spa === null) {
            return null;
        }

        $apiResult = $this->api->handle($method, $path, $body);
        if (($apiResult['status'] ?? 0) !== 404) {
            return $apiResult;
        }

        return $this->spa->handle($method, $path);
    }

    public function stream(callable $write): void
    {
        $this->api?->stream($write);
    }
}
