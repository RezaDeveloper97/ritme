<?php

declare(strict_types=1);

namespace App\Filament\Pages\Settings;

use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Forms\Components\Field;
use Filament\Notifications\Notification;
use Filament\Pages\Page;
use Filament\Schemas\Components\Actions;
use Filament\Schemas\Components\Component;
use Filament\Schemas\Components\EmbeddedSchema;
use Filament\Schemas\Components\Form;
use Filament\Schemas\Schema;
use UnitEnum;

/**
 * One settings group (L1-01 DTOs) as a Filament form page. Reads through SettingsRepository, writes through the
 * UpdateSettings action (SettingObserver bumps `settings`, `seo` and `pages` after commit) and records the change in
 * the activity log.
 *
 * @property-read Schema $form
 */
abstract class SettingsPage extends Page
{
    /**
     * @var array<string, mixed>|null
     */
    public ?array $data = [];

    protected static string|UnitEnum|null $navigationGroup = 'تنظیمات';

    /**
     * Roles (besides super-admin) that may edit this group.
     *
     * @var list<AdminRole>
     */
    protected static array $roles = [AdminRole::Editor];

    abstract protected static function group(): SettingGroup;

    /**
     * @return array<Component|Field>
     */
    abstract protected function fields(): array;

    public static function canAccess(): bool
    {
        return AdminAccess::allows(Filament::auth()->user(), static::$roles);
    }

    public function mount(SettingsRepository $settings): void
    {
        $this->form->fill($settings->group(static::group())->toArray());
    }

    public function form(Schema $schema): Schema
    {
        return $schema->components($this->fields())->statePath('data');
    }

    public function content(Schema $schema): Schema
    {
        return $schema->components([
            Form::make([EmbeddedSchema::make('form')])
                ->id('form')
                ->livewireSubmitHandler('save')
                ->footer([
                    Actions::make([
                        Action::make('save')->label('ذخیره')->submit('save')->keyBindings(['mod+s']),
                    ])->key('form-actions'),
                ]),
        ]);
    }

    public function save(UpdateSettings $update, SettingsRepository $settings): void
    {
        abort_unless(static::canAccess(), 403);

        $group = static::group();
        /** @var array<string, mixed> $state */
        $state = $this->form->getState();
        $old = $settings->group($group)->toArray();

        $new = $update->handle($group, $state)->toArray();

        $changedKeys = array_keys(array_filter(
            $new,
            static fn (mixed $value, string $key): bool => ($old[$key] ?? null) !== $value,
            ARRAY_FILTER_USE_BOTH,
        ));
        if ($changedKeys !== []) {
            activity('settings')
                ->causedBy(Filament::auth()->user())
                ->event('updated')
                ->withProperties([
                    'group' => $group->value,
                    'attributes' => array_intersect_key($new, array_flip($changedKeys)),
                    'old' => array_intersect_key($old, array_flip($changedKeys)),
                ])
                ->log("settings.{$group->value}");
        }

        $this->form->fill($new);

        Notification::make()->success()->title('تنظیمات ذخیره شد.')->send();
    }
}
