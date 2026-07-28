<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core;

final class Entry
{
    /** @param list<string> $tags */
    public function __construct(
        public string $id,
        public string $batchId,
        public string $type,
        public array $content,
        public string $createdAt,
        public string $requestId = '',
        public string $correlationId = '',
        public array $tags = [],
    ) {
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        $data = [
            'id' => $this->id,
            'batch_id' => $this->batchId,
            'type' => $this->type,
            'content' => $this->content,
            'created_at' => $this->createdAt,
        ];

        if ($this->requestId !== '') {
            $data['request_id'] = $this->requestId;
        }
        if ($this->correlationId !== '') {
            $data['correlation_id'] = $this->correlationId;
        }
        if ($this->tags !== []) {
            $data['tags'] = $this->tags;
        }

        return $data;
    }
}
