<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Laravel\Support;

use Illuminate\Http\Request;

final class RequestContext
{
    public const BATCH_ATTRIBUTE = 'microscope.batch_id';

    public static function start(Request $request): string
    {
        $batchId = bin2hex(random_bytes(16));
        $request->attributes->set(self::BATCH_ATTRIBUTE, $batchId);

        return $batchId;
    }

    public static function batchId(?Request $request = null): ?string
    {
        $request ??= request();
        if ($request === null) {
            return null;
        }

        $batchId = $request->attributes->get(self::BATCH_ATTRIBUTE);

        return is_string($batchId) && $batchId !== '' ? $batchId : null;
    }
}
