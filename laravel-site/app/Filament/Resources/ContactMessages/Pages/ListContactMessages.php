<?php

declare(strict_types=1);

namespace App\Filament\Resources\ContactMessages\Pages;

use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Filament\Resources\ContactMessages\ContactMessageResource;
use Filament\Resources\Pages\ListRecords;
use Filament\Schemas\Components\Tabs\Tab;
use Illuminate\Database\Eloquent\Builder;

final class ListContactMessages extends ListRecords
{
    protected static string $resource = ContactMessageResource::class;

    protected function getHeaderActions(): array
    {
        return [ContactMessageResource::exportAction()];
    }

    /**
     * @return array<string, Tab>
     */
    public function getTabs(): array
    {
        $tabs = [];
        foreach (ContactMessageStatus::cases() as $status) {
            $tabs[$status->value] = Tab::make($status->label())
                ->modifyQueryUsing(static fn (Builder $query): Builder => $query->where('status', $status->value));
        }
        $tabs['all'] = Tab::make('همه');

        return $tabs;
    }

    public function getDefaultActiveTab(): string
    {
        return ContactMessageStatus::Unread->value;
    }

    /**
     * The current tab, topic filter and search, passed on to the export link.
     *
     * @return array<string, string>
     */
    public function exportParameters(): array
    {
        $status = ContactMessageStatus::tryFrom((string) $this->activeTab);
        $topic = $this->tableFilters['topic']['value'] ?? null;
        $search = trim((string) $this->tableSearch);

        return array_filter([
            'status' => $status?->value,
            'topic' => is_string($topic) && $topic !== '' ? $topic : null,
            'search' => $search !== '' ? $search : null,
        ], static fn (?string $value): bool => $value !== null);
    }
}
