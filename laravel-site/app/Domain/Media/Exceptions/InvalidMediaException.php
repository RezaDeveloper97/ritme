<?php

declare(strict_types=1);

namespace App\Domain\Media\Exceptions;

use RuntimeException;

/**
 * An upload the pipeline refuses. The message is admin-facing (Persian); `reason` is a stable code for callers/tests.
 */
final class InvalidMediaException extends RuntimeException
{
    public function __construct(public readonly string $reason, string $message)
    {
        parent::__construct($message);
    }

    public static function unreadable(): self
    {
        return new self('unreadable', 'فایل خوانده نشد.');
    }

    public static function tooLarge(int $maxBytes): self
    {
        return new self('too_large', sprintf('حجم فایل بیشتر از %d مگابایت است.', intdiv($maxBytes, 1024 * 1024)));
    }

    public static function unsupportedType(string $mime): self
    {
        return new self('unsupported_type', sprintf('نوع فایل «%s» پشتیبانی نمی‌شود.', $mime));
    }

    public static function tooManyPixels(int $maxDimension): self
    {
        return new self('too_many_pixels', sprintf('ابعاد تصویر نباید از %d پیکسل بیشتر باشد.', $maxDimension));
    }

    public static function unsafeSvg(): self
    {
        return new self('unsafe_svg', 'فایل SVG معتبر نیست یا پاک‌سازی نشد.');
    }

    public static function corrupt(): self
    {
        return new self('corrupt', 'تصویر خراب است یا قابل پردازش نیست.');
    }
}
