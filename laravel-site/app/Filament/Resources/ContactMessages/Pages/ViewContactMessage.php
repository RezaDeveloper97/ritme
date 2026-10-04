<?php

declare(strict_types=1);

namespace App\Filament\Resources\ContactMessages\Pages;

use App\Domain\Contact\Actions\ChangeContactMessageStatus;
use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Models\ContactMessage;
use App\Filament\Resources\ContactMessages\ContactMessageResource;
use Filament\Actions\Action;
use Filament\Actions\DeleteAction;
use Filament\Facades\Filament;
use Filament\Resources\Pages\ViewRecord;
use Filament\Support\Icons\Heroicon;

/**
 * One message. Opening it marks an unread message read (not activity-logged); header actions reply by mail/phone,
 * mark it unread again, archive / restore it, or delete it.
 */
final class ViewContactMessage extends ViewRecord
{
    protected static string $resource = ContactMessageResource::class;

    public function mount(int|string $record): void
    {
        parent::mount($record);

        $message = $this->getRecord();
        if ($message instanceof ContactMessage && $message->status === ContactMessageStatus::Unread) {
            app(ChangeContactMessageStatus::class)->handle($message, ContactMessageStatus::Read, Filament::auth()->user(), log: false);
        }
    }

    protected function getHeaderActions(): array
    {
        $record = $this->getRecord();
        $message = $record instanceof ContactMessage ? $record : null;

        return [
            Action::make('reply')
                ->label($message?->email !== null ? 'پاسخ با ایمیل' : 'تماس تلفنی')
                ->icon($message?->email !== null ? Heroicon::OutlinedPaperAirplane : Heroicon::OutlinedPhone)
                ->url($message !== null ? ContactMessageResource::replyUrl($message) : null)
                ->visible($message !== null && ContactMessageResource::replyUrl($message) !== null),
            ContactMessageResource::statusAction('markUnread', ContactMessageStatus::Unread, 'خوانده‌نشده', Heroicon::OutlinedEnvelope)
                ->visible(static fn (ContactMessage $record): bool => $record->status === ContactMessageStatus::Read),
            ContactMessageResource::statusAction('archive', ContactMessageStatus::Archived, 'بایگانی', Heroicon::OutlinedArchiveBox)
                ->visible(static fn (ContactMessage $record): bool => $record->status !== ContactMessageStatus::Archived),
            ContactMessageResource::statusAction('unarchive', ContactMessageStatus::Read, 'خروج از بایگانی', Heroicon::OutlinedArrowUturnLeft)
                ->visible(static fn (ContactMessage $record): bool => $record->status === ContactMessageStatus::Archived),
            DeleteAction::make(),
        ];
    }
}
