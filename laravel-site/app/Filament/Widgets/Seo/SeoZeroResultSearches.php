<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Seo;

use App\Domain\Seo\Audit\Queries\ZeroResultSearches;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Filament\Widgets\TableWidget;

/**
 * Site searches that found nothing in the last 7 days (anonymised, L4-04) — ideas for content and synonyms.
 */
final class SeoZeroResultSearches extends TableWidget
{
    use SeoWidgetAccess;

    protected static ?int $sort = 14;

    public function table(Table $table): Table
    {
        return $table
            ->heading('جستجوهای بی‌نتیجه (۷ روز)')
            ->records(static function (): array {
                $rows = [];
                foreach (app(ZeroResultSearches::class)->top() as $row) {
                    $rows[md5($row['term'])] = $row;
                }

                return $rows;
            })
            ->paginated(false)
            ->emptyStateHeading('جستجوی بی‌نتیجه‌ای ثبت نشده است.')
            ->columns([
                TextColumn::make('term')->label('عبارت'),
                TextColumn::make('count')->label('دفعات')->formatStateUsing(static fn (int $state): string => fa_digits($state)),
            ]);
    }
}
