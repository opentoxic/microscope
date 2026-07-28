<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Http;

use Opentoxic\Microscope\Adaptor\Php\Core\EntryDetailBuilder;
use Opentoxic\Microscope\Adaptor\Php\Core\EntryType;
use Opentoxic\Microscope\Adaptor\Php\Core\Hub;

final class ApiRouter
{
    public function __construct(private readonly Hub $hub)
    {
    }

    /** @return array{status: int, headers: array<string, string>, body: string} */
    public function handle(string $method, string $path, string $body = ''): array
    {
        $prefix = rtrim($this->hub->config()->pathPrefix(), '/');

        if ($path === $prefix . '/api/entries' && $method === 'GET') {
            return $this->json(200, $this->hub->store()->list(
                $_GET['type'] ?? null,
                $_GET['search'] ?? null,
                isset($_GET['limit']) ? (int) $_GET['limit'] : 50,
                isset($_GET['offset']) ? (int) $_GET['offset'] : 0,
            ));
        }

        if ($path === $prefix . '/api/entries' && $method === 'POST') {
            return $this->createCustom($body);
        }

        if (preg_match('#^' . preg_quote($prefix, '#') . '/api/entries/([^/]+)$#', $path, $m) && $method === 'GET') {
            return $this->getEntry($m[1]);
        }

        if ($path === $prefix . '/api/stream' && $method === 'GET') {
            return ['status' => 200, 'headers' => ['Content-Type' => 'text/event-stream'], 'body' => '', 'stream' => true];
        }

        if ($path === $prefix . '/api/prune' && $method === 'POST') {
            $deleted = $this->hub->store()->clearAll();

            return $this->json(200, ['deleted' => $deleted]);
        }

        if ($path === $prefix . '/api/storage' && $method === 'GET') {
            return $this->json(200, $this->hub->store()->storageUsage());
        }

        if ($path === $prefix . '/api/recording' && $method === 'GET') {
            return $this->json(200, ['paused' => $this->hub->recordingPaused()]);
        }

        if ($path === $prefix . '/api/recording' && $method === 'PUT') {
            return $this->setRecording($body);
        }

        if ($path === $prefix . '/api/redaction' && $method === 'GET') {
            return $this->json(200, ['enabled' => $this->hub->redactSensitive()]);
        }

        if ($path === $prefix . '/api/redaction' && $method === 'PUT') {
            return $this->setRedaction($body);
        }

        if ($path === $prefix . '/api/settings' && $method === 'GET') {
            return $this->json(200, ['settings' => $this->hub->typeSettings()]);
        }

        if (preg_match('#^' . preg_quote($prefix, '#') . '/api/settings/([^/]+)$#', $path, $m) && $method === 'PUT') {
            return $this->updateSetting($m[1], $body);
        }

        return $this->json(404, ['error' => 'not found']);
    }

    public function stream(callable $write): void
    {
        $write(": connected\n\n");
        $this->hub->subscribe(function ($entry) use ($write): void {
            $payload = json_encode($entry->toArray(), JSON_THROW_ON_ERROR);
            $write("id: {$entry->id}\nevent: entry\ndata: {$payload}\n\n");
        });
        $this->hub->subscribeControl(function (array $event) use ($write): void {
            $payload = json_encode($event, JSON_THROW_ON_ERROR);
            $write("event: control\ndata: {$payload}\n\n");
        });

        while (! connection_aborted()) {
            $write(": heartbeat\n\n");
            sleep(20);
        }
    }

    private function createCustom(string $body): array
    {
        if ($this->hub->recordingPaused()) {
            return $this->json(409, ['error' => 'recording is paused']);
        }
        $input = json_decode($body, true);
        if (! is_array($input)) {
            return $this->json(400, ['error' => 'invalid JSON body']);
        }
        $name = trim((string) ($input['name'] ?? ''));
        if ($name === '' || strlen($name) > 120) {
            return $this->json(422, ['error' => 'name must contain 1 to 120 characters']);
        }
        $content = is_array($input['content'] ?? null) ? $input['content'] : [];
        $content['name'] = $name;
        $id = $this->hub->record(EntryType::CUSTOM, $content, null, null);

        return $this->json(202, ['id' => $id]);
    }

    private function getEntry(string $id): array
    {
        $entry = $this->hub->store()->get($id);
        if ($entry === null) {
            return $this->json(404, ['error' => 'entry not found']);
        }
        $batch = $this->hub->store()->listByBatch($entry->batchId);

        return $this->json(200, EntryDetailBuilder::build($entry, $batch));
    }

    private function setRecording(string $body): array
    {
        $input = json_decode($body, true);
        if (! is_array($input) || ! isset($input['paused']) || ! is_bool($input['paused'])) {
            return $this->json(400, ['error' => 'paused must be a boolean']);
        }
        $this->hub->setRecordingPaused($input['paused']);

        return $this->json(200, ['paused' => $this->hub->recordingPaused()]);
    }

    private function setRedaction(string $body): array
    {
        $input = json_decode($body, true);
        if (! is_array($input) || ! isset($input['enabled']) || ! is_bool($input['enabled'])) {
            return $this->json(400, ['error' => 'enabled must be a boolean']);
        }
        $this->hub->setRedactSensitive($input['enabled']);

        return $this->json(200, ['enabled' => $this->hub->redactSensitive()]);
    }

    private function updateSetting(string $type, string $body): array
    {
        if (! EntryType::isValid($type)) {
            return $this->json(404, ['error' => 'unknown signal type']);
        }
        $input = json_decode($body, true);
        if (! is_array($input) || ! isset($input['enabled']) || ! is_bool($input['enabled'])) {
            return $this->json(400, ['error' => 'enabled must be a boolean']);
        }
        $deleted = $this->hub->setTypeEnabled($type, $input['enabled']);

        return $this->json(200, ['type' => $type, 'enabled' => $input['enabled'], 'deleted' => $deleted]);
    }

    /** @param array<string, mixed> $data */
    private function json(int $status, array $data): array
    {
        return [
            'status' => $status,
            'headers' => ['Content-Type' => 'application/json'],
            'body' => json_encode($data, JSON_THROW_ON_ERROR),
        ];
    }
}
