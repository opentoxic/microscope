<?php

declare(strict_types=1);

namespace Opentoxic\Microscope\Adaptor\Php\Core;

final class EntryType
{
    public const REQUEST = 'request';
    public const QUERY = 'query';
    public const LOG = 'log';
    public const EVENT = 'event';
    public const NOTIFICATION = 'notification';
    public const EXCEPTION = 'exception';
    public const CACHE = 'cache';
    public const REDIS = 'redis';
    public const JOB = 'job';
    public const SCHEDULE = 'schedule';
    public const MAIL = 'mail';
    public const HTTP_CLIENT = 'http-client';
    public const WEBSOCKET = 'websocket';
    public const PERFORMANCE = 'performance';
    public const METRIC = 'metric';
    public const CUSTOM = 'custom';
    public const TOPIC = 'topic';

    /** @var list<string> */
    public const ALL = [
        self::REQUEST,
        self::QUERY,
        self::LOG,
        self::EVENT,
        self::NOTIFICATION,
        self::EXCEPTION,
        self::CACHE,
        self::REDIS,
        self::JOB,
        self::SCHEDULE,
        self::MAIL,
        self::HTTP_CLIENT,
        self::WEBSOCKET,
        self::PERFORMANCE,
        self::METRIC,
        self::CUSTOM,
        self::TOPIC,
    ];

    public static function isValid(string $type): bool
    {
        return in_array($type, self::ALL, true);
    }
}
