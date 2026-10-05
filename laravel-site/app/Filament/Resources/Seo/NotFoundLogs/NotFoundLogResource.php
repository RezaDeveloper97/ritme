<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\NotFoundLogs;

use App\Domain\Seo\Redirects\Actions\CreateRedirectFromNotFound;
use App\Domain\Seo\Redirects\Enums\AgentClass;
use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Filament\Resources\Seo\NotFoundLogs\Pages\ListNotFoundLogs;
use App\Filament\Resources\Seo\Redirects\RedirectForm;
use App\Filament\Resources\Seo\Redirects\RedirectResource;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Actions\DeleteAction;
use Filament\Actions\DeleteBulkAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TextInput;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\SelectFilter;
use Filament\Tables\Table;
use Illuminate\Support\Carbon;
use UnitEnum;

/**
 * 404 monitor (L7-03): aggregated not-found paths (hits, last seen, last referer, agent class — no personal data),
 * most-hit first, with a "create redirect" action (CreateRedirectFromNotFound: SaveRedirect rules, the row is then
 * removed). Rows older than 90 days are purged by the scheduler. Access: NotFoundLogPolicy (SEO manager + super-admin).
 */
final class NotFoundLogResource extends Resource
{
    protected static ?string $model = NotFoundLog::class;

    protected static ?string $slug = 'seo/not-found';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedExclamationTriangle;

    protected static string|UnitEnum|null $navigationGroup = 'سئو';

    protected static ?int $navigationSort = 41;

    protected static ?string $navigationLabel = 'پایش ۴۰۴';

    protected static ?string $modelLabel = 'آدرس ۴۰۴';

    protected static ?string $pluralModelLabel = 'آدرس‌های ۴۰۴';

    public static function table(Table $table): Table
    {
        $date = static fn (?Carbon $state): ?string => $state ? jdate($state, 'Y/m/d H:i') : null;

        return $table
            ->defaultSort('hits', 'desc')
            ->columns([
                TextColumn::make('path')->label('آدرس')->searchable()->limit(70)->tooltip(static fn (NotFoundLog $r): string => $r->path)
                    ->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('hits')->label('تعداد')->numeric()->sortable(),
                TextColumn::make('last_seen_at')->label('آخرین بار')->formatStateUsing($date)->sortable(),
                TextColumn::make('first_seen_at')->label('اولین بار')->formatStateUsing($date)->sortable()->toggleable(isToggledHiddenByDefault: true),
                TextColumn::make('referer')->label('ارجاع‌دهنده')->limit(50)->placeholder('—')->extraAttributes(['dir' => 'ltr']),
                TextColumn::make('agent')->label('نوع')->badge()
                    ->formatStateUsing(static fn (AgentClass $state): string => $state->label())
                    ->color(static fn (AgentClass $state): string => $state === AgentClass::Bot ? 'gray' : 'info'),
            ])
            ->filters([
                SelectFilter::make('agent')->label('نوع')->options([AgentClass::Human->value => AgentClass::Human->label(), AgentClass::Bot->value => AgentClass::Bot->label()]),
            ])
            ->recordActions([
                self::createRedirectAction(),
                DeleteAction::make(),
            ])
            ->toolbarActions([DeleteBulkAction::make()]);
    }

    private static function createRedirectAction(): Action
    {
        return Action::make('createRedirect')
            ->label('ساخت ریدایرکت')
            ->icon(Heroicon::OutlinedArrowUturnRight)
            ->authorize(static fn (): bool => RedirectResource::allows('create', Redirect::class))
            ->modalHeading(static fn (NotFoundLog $record): string => 'ریدایرکت برای '.$record->path)
            ->schema([
                Select::make('code')->label('کد وضعیت')->options(RedirectCode::options())->default(RedirectCode::Permanent->value)
                    ->required()->live()->native(false),
                TextInput::make('to_url')->label('مقصد')->maxLength(2048)->extraInputAttributes(['dir' => 'ltr'])
                    ->placeholder('/blog')
                    ->hidden(static fn (Get $get): bool => (int) $get('code') === RedirectCode::Gone->value)
                    ->required(static fn (Get $get): bool => (int) $get('code') !== RedirectCode::Gone->value),
            ])
            ->action(static function (NotFoundLog $record, array $data, CreateRedirectFromNotFound $create): void {
                try {
                    $create->handle($record, (string) ($data['to_url'] ?? ''), (int) ($data['code'] ?? 301), Filament::auth()->user());
                } catch (InvalidRedirect $e) {
                    throw RedirectForm::fail($e, 'mountedActions.0.data.');
                }
            })
            ->successNotificationTitle('ریدایرکت ساخته شد');
    }

    public static function getPages(): array
    {
        return [
            'index' => ListNotFoundLogs::route('/'),
        ];
    }
}
