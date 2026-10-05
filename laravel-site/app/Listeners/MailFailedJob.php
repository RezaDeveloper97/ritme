<?php

declare(strict_types=1);

namespace App\Listeners;

use Illuminate\Contracts\Cache\Repository as Cache;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Mail\Mailer;
use Illuminate\Mail\Message;
use Illuminate\Queue\Events\JobFailed;
use Psr\Log\LoggerInterface;
use Throwable;

/**
 * L10-02: mails the operator (`logging.alerts.mail_to`, OPS_ALERT_EMAIL) when a queued job fails for good — on cPanel
 * nobody watches a worker console, so a dead mail/SMS/optimisation job would otherwise go unnoticed.
 *
 * - Synchronous on purpose (never queued): the queue may be exactly what is broken.
 * - At most one mail per job class per `logging.alerts.throttle_minutes` (cache `add`), so a broken SMTP server or a
 *   bad release cannot flood the mailbox; every failure is still in the log and in `php artisan queue:failed`.
 * - No payload (orders, bookings and contact messages carry personal data); e-mail addresses and phone-like digit
 *   runs in the exception message are masked.
 * - Never throws: a failing alert must not hide the original failure.
 *
 * Auto-discovered (app/Listeners, Laravel event discovery; cached by `optimize`).
 */
final class MailFailedJob
{
    public const THROTTLE_KEY = 'ops-alert:job-failed:';

    public function __construct(
        private readonly Config $config,
        private readonly Cache $cache,
        private readonly Mailer $mailer,
        private readonly LoggerInterface $log,
    ) {}

    public function handle(JobFailed $event): void
    {
        $to = $this->config->get('logging.alerts.mail_to');
        if (! is_string($to) || filter_var($to, FILTER_VALIDATE_EMAIL) === false) {
            return;
        }

        $job = $event->job->resolveName();
        $minutes = max(1, (int) $this->config->get('logging.alerts.throttle_minutes', 60));

        try {
            if (! $this->cache->add(self::THROTTLE_KEY.sha1($job), 1, $minutes * 60)) {
                return;
            }

            $app = (string) $this->config->get('app.name', 'Ritme');
            $host = (string) parse_url((string) $this->config->get('app.url', ''), PHP_URL_HOST);
            $body = implode("\n", [
                'A queued job failed after all its attempts.',
                '',
                'Site: '.((string) $this->config->get('app.url', '')),
                'Job: '.$job,
                'Queue: '.$event->connectionName.' / '.$event->job->getQueue(),
                'Attempts: '.$event->job->attempts(),
                'Error: '.$event->exception::class.': '.self::mask($event->exception->getMessage()),
                'Time: '.now()->toDateTimeString().' ('.date_default_timezone_get().')',
                '',
                'Details: storage/logs/laravel-<date>.log on the host.',
                'Retry: php artisan queue:retry all   ·   list: php artisan queue:failed',
                "Further failures of this job are not mailed for {$minutes} minutes.",
            ]);

            $this->mailer->raw($body, static function (Message $message) use ($to, $app, $host, $job): void {
                $message->to($to)->subject("[{$app}] Job failed on {$host}: ".class_basename($job));
            });
        } catch (Throwable $e) {
            $this->log->warning('Job-failure alert mail could not be sent: '.$e->getMessage());
        }
    }

    /** Masks e-mail addresses and 7+ digit runs (mobiles, codes) and caps the length. */
    public static function mask(string $message): string
    {
        $message = (string) preg_replace('/[^\s@<>"\']+@[^\s@<>"\']+/u', '[email]', $message);
        $message = (string) preg_replace('/(?:\+?\d[\d\s-]{5,}\d)/u', '[number]', $message);

        return mb_strimwidth($message, 0, 500, '…');
    }
}
