<?php

declare(strict_types=1);

namespace App\Domain\Contact\Support;

use App\Domain\Contact\Enums\FormTimerResult;
use Illuminate\Contracts\Encryption\DecryptException;
use Illuminate\Contracts\Encryption\Encrypter;

/**
 * Time trap for the contact form (no captcha): the page embeds an encrypted render time; a submission that comes back
 * faster than MIN_SECONDS (bots post instantly), or with a missing / tampered token, is spam and dropped silently.
 * An honest visitor whose form is older than MAX_SECONDS gets a «send again» validation message instead.
 *
 * The token is minted on every render, so the contact page opts out of the full-page cache (ContactController):
 * a cached page would carry the cache-store time and make the trap useless.
 */
final readonly class FormTimer
{
    public const MIN_SECONDS = 3;

    public const MAX_SECONDS = 86400;

    private const PREFIX = 'contact-form|';

    public function __construct(private Encrypter $encrypter) {}

    public function issue(): string
    {
        return $this->encrypter->encryptString(self::PREFIX.now()->getTimestamp());
    }

    public function check(?string $token): FormTimerResult
    {
        if ($token === null || $token === '' || strlen($token) > 1024) {
            return FormTimerResult::Invalid;
        }

        try {
            $plain = $this->encrypter->decryptString($token);
        } catch (DecryptException) {
            return FormTimerResult::Invalid;
        }

        if (! str_starts_with($plain, self::PREFIX) || ! ctype_digit($issued = substr($plain, strlen(self::PREFIX)))) {
            return FormTimerResult::Invalid;
        }

        $elapsed = now()->getTimestamp() - (int) $issued;

        return match (true) {
            $elapsed < self::MIN_SECONDS => FormTimerResult::TooFast,
            $elapsed > self::MAX_SECONDS => FormTimerResult::Expired,
            default => FormTimerResult::Ok,
        };
    }
}
