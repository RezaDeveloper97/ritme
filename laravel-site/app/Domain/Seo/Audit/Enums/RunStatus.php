<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Enums;

enum RunStatus: string
{
    case Queued = 'queued';
    case Running = 'running';
    case Completed = 'completed';
    case Failed = 'failed';

    public function label(): string
    {
        return match ($this) {
            self::Queued => 'در صف',
            self::Running => 'در حال اجرا',
            self::Completed => 'انجام شد',
            self::Failed => 'ناموفق',
        };
    }

    public function color(): string
    {
        return match ($this) {
            self::Queued, self::Running => 'info',
            self::Completed => 'success',
            self::Failed => 'danger',
        };
    }
}
