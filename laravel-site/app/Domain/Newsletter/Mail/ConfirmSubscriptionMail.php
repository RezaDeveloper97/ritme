<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Mail;

use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Mail\Mailable;
use Illuminate\Mail\Mailables\Content;
use Illuminate\Mail\Mailables\Envelope;
use Illuminate\Mail\Mailables\Headers;

/**
 * Double opt-in mail: one confirm link plus the unsubscribe link (also as List-Unsubscribe one-click headers).
 * Queued after commit so the signup response time doesn't reveal whether an address was already subscribed (F16).
 */
final class ConfirmSubscriptionMail extends Mailable implements ShouldQueue
{
    use Queueable;

    public function __construct(public readonly string $confirmUrl, public readonly string $unsubscribeUrl)
    {
        $this->afterCommit();
    }

    public function envelope(): Envelope
    {
        return new Envelope(subject: (string) __('blog.newsletter.mail.subject'));
    }

    public function content(): Content
    {
        return new Content(
            html: 'pages.blog.newsletter.mail.confirm',
            text: 'pages.blog.newsletter.mail.confirm-text',
        );
    }

    public function headers(): Headers
    {
        return new Headers(text: [
            'List-Unsubscribe' => '<'.$this->unsubscribeUrl.'>',
            'List-Unsubscribe-Post' => 'List-Unsubscribe=One-Click',
        ]);
    }
}
