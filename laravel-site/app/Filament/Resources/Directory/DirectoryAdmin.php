<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory;

use App\Domain\Directory\Enums\Weekday;
use App\View\Components\Icon;
use DateTimeInterface;
use Filament\Facades\Filament;
use Filament\Forms\Components\Repeater;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TextInput;
use Filament\Schemas\Components\Fieldset;
use Filament\Schemas\Components\Group;
use Filament\Schemas\Components\Utilities\Get;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\Gate;

/**
 * Shared bits of the directory admin (L5-06): navigation group, Jalali date formatting, the sprite icon picker and
 * the weekly opening-hours editor (form state ⇄ the `opening_hours` JSON of places / join requests).
 */
final class DirectoryAdmin
{
    public const NAV_GROUP = 'راهنمای مادر و کودک';

    private const TIME = '/^([01]?\d|2[0-3]):[0-5]\d$/';

    public static function user(): ?Model
    {
        $user = Filament::auth()->user();

        return $user instanceof Model ? $user : null;
    }

    public static function can(string $ability, mixed $subject): bool
    {
        return Gate::forUser(Filament::auth()->user())->allows($ability, $subject);
    }

    public static function date(mixed $state, string $format = 'Y/m/d H:i'): ?string
    {
        return $state instanceof DateTimeInterface ? jdate($state, $format) : null;
    }

    /**
     * A select of the local SVG sprite icons (resources/svg/icons) with a preview of each.
     */
    public static function iconSelect(string $name = 'icon'): Select
    {
        return Select::make($name)
            ->label('آیکن')
            ->native(false)
            ->searchable()
            ->allowHtml()
            ->placeholder('بدون آیکن')
            ->options(static function (): array {
                $options = [];
                foreach (Icon::names() as $icon) {
                    $options[$icon] = self::iconLabel($icon);
                }

                return $options;
            })
            ->getSearchResultsUsing(static function (string $search): array {
                $options = [];
                foreach (Icon::names() as $icon) {
                    if (str_contains($icon, strtolower(trim($search)))) {
                        $options[$icon] = self::iconLabel($icon);
                    }
                }

                return $options;
            })
            ->in(static fn (): array => Icon::names());
    }

    /**
     * The weekly hours editor: per weekday «نامشخص / تعطیل / باز» and, when open, one or more HH:MM ranges (a range
     * ending before it starts runs past midnight). Lives under the `hours` state path; convert with toOpeningHours().
     */
    public static function hoursEditor(): Group
    {
        $days = [];
        foreach (Weekday::cases() as $day) {
            $days[] = Fieldset::make($day->label())->statePath($day->value)->columns(1)->schema([
                Select::make('state')
                    ->hiddenLabel()
                    ->options(['unknown' => 'نامشخص', 'closed' => 'تعطیل', 'open' => 'باز'])
                    ->default('unknown')
                    ->selectablePlaceholder(false)
                    ->live(),
                Repeater::make('ranges')
                    ->hiddenLabel()
                    ->visible(static fn (Get $get): bool => $get('state') === 'open')
                    ->schema([
                        TextInput::make('opens')->label('از')->placeholder('09:00')->extraInputAttributes(['dir' => 'ltr'])
                            ->required()->regex(self::TIME)->validationMessages(['regex' => 'ساعت را به شکل ۰۹:۰۰ بنویسید.']),
                        TextInput::make('closes')->label('تا')->placeholder('20:00')->extraInputAttributes(['dir' => 'ltr'])
                            ->required()->regex(self::TIME)->validationMessages(['regex' => 'ساعت را به شکل ۲۰:۰۰ بنویسید.']),
                    ])
                    ->columns(2)
                    ->minItems(1)
                    ->maxItems(4)
                    ->defaultItems(1)
                    ->reorderable(false)
                    ->addActionLabel('بازه دیگر'),
            ]);
        }

        return Group::make($days)->statePath('hours')->columns(['default' => 1, 'lg' => 2]);
    }

    /**
     * `opening_hours` JSON → editor state.
     *
     * @return array<string, array{state: string, ranges: list<array{opens: string, closes: string}>}>
     */
    public static function hoursState(mixed $openingHours): array
    {
        $hours = is_array($openingHours) ? $openingHours : [];
        $state = [];
        foreach (Weekday::cases() as $day) {
            if (! array_key_exists($day->value, $hours)) {
                $state[$day->value] = ['state' => 'unknown', 'ranges' => []];

                continue;
            }

            $ranges = [];
            foreach ((array) $hours[$day->value] as $range) {
                if (is_array($range) && isset($range['opens'], $range['closes'])) {
                    $ranges[] = ['opens' => (string) $range['opens'], 'closes' => (string) $range['closes']];
                }
            }
            $state[$day->value] = ['state' => $ranges === [] ? 'closed' : 'open', 'ranges' => $ranges];
        }

        return $state;
    }

    /**
     * Editor state → `opening_hours` JSON (null when every day is unknown). PlaceObserver normalises it further.
     *
     * @return array<string, list<array{opens: string, closes: string}>>|null
     */
    public static function toOpeningHours(mixed $state): ?array
    {
        $hours = [];
        foreach (Weekday::cases() as $day) {
            $entry = is_array($state) && is_array($state[$day->value] ?? null) ? $state[$day->value] : [];
            $mode = $entry['state'] ?? 'unknown';
            if ($mode === 'closed') {
                $hours[$day->value] = [];
            } elseif ($mode === 'open') {
                $ranges = [];
                foreach ((array) ($entry['ranges'] ?? []) as $range) {
                    $opens = trim((string) (is_array($range) ? ($range['opens'] ?? '') : ''));
                    $closes = trim((string) (is_array($range) ? ($range['closes'] ?? '') : ''));
                    if (preg_match(self::TIME, $opens) === 1 && preg_match(self::TIME, $closes) === 1) {
                        $ranges[] = ['opens' => $opens, 'closes' => $closes];
                    }
                }
                $hours[$day->value] = $ranges;
            }
        }

        return $hours === [] ? null : $hours;
    }

    private static function iconLabel(string $icon): string
    {
        $svg = (string) @file_get_contents(Icon::directory().'/'.$icon.'.svg');
        $svg = preg_replace('/<svg\b/', '<svg width="20" height="20" class="size-5 shrink-0" aria-hidden="true"', $svg, 1) ?? '';

        return '<span class="flex items-center gap-2">'.$svg.'<span dir="ltr">'.e($icon).'</span></span>';
    }
}
