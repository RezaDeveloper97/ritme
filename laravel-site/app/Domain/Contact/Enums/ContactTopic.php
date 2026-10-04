<?php

declare(strict_types=1);

namespace App\Domain\Contact\Enums;

/**
 * Contact form topics (design/html/contact.html chips), in display order. The topic also picks the mailbox that is
 * notified (ContactRecipients): partnership → partnership email, privacy → data-protection email, else support.
 */
enum ContactTopic: string
{
    case Support = 'support';
    case Partnership = 'partnership';
    case Professionals = 'professionals';
    case Media = 'media';
    case Privacy = 'privacy';

    public function label(): string
    {
        return match ($this) {
            self::Support => 'پشتیبانی کاربر',
            self::Partnership => 'همکاری کسب‌وکار',
            self::Professionals => 'متخصصان سلامت',
            self::Media => 'رسانه',
            self::Privacy => 'حریم خصوصی و داده',
        };
    }

    /**
     * @return array<string, string> value => Persian label
     */
    public static function options(): array
    {
        $options = [];
        foreach (self::cases() as $case) {
            $options[$case->value] = $case->label();
        }

        return $options;
    }
}
