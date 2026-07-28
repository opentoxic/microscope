<?php

return [
    'enabled' => env('MICROSCOPE_ENABLED', false),
    'path' => env('MICROSCOPE_PATH', 'microscope'),
    'max_body_bytes' => (int) env('MICROSCOPE_MAX_BODY_BYTES', 65536),
    'redact_sensitive' => filter_var(env('MICROSCOPE_REDACT_SENSITIVE', false), FILTER_VALIDATE_BOOL),
    'middleware' => ['web'],
];
