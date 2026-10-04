<?php

declare(strict_types=1);

use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Models\ContactMessage;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\ContactMessages\ContactMessageResource;
use App\Filament\Resources\ContactMessages\Pages\ListContactMessages;
use App\Filament\Resources\ContactMessages\Pages\ViewContactMessage;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Illuminate\Support\Carbon;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;

function inboxAdmin(?AdminRole $role = AdminRole::Support): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return $user;
}

/**
 * @param  array<string, mixed>  $attributes
 */
function inboxMessage(array $attributes = []): ContactMessage
{
    $message = ContactMessage::query()->create([
        'topic' => ContactTopic::Support,
        'name' => 'مریم',
        'email' => 'maryam@example.com',
        'message' => 'سلام، یادآور برایم نمی‌آید.',
        ...$attributes,
    ]);
    $message->forceFill(['created_at' => Carbon::parse('2026-03-21 08:30:00', 'Asia/Tehran')])->save();

    return $message;
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');

    $this->unread = inboxMessage();
    $this->read = inboxMessage(['name' => 'سارا', 'email' => null, 'phone' => '09123456789', 'topic' => ContactTopic::Media, 'status' => ContactMessageStatus::Read, 'read_at' => now()]);
    $this->archived = inboxMessage(['name' => 'نگار', 'email' => 'negar@example.com', 'status' => ContactMessageStatus::Archived, 'read_at' => now(), 'message' => '=HYPERLINK("x")']);
});

it('lets the support role and super-admins use the inbox and denies everyone else', function (): void {
    $index = ContactMessageResource::getUrl('index');
    $view = ContactMessageResource::getUrl('view', ['record' => $this->unread]);
    $export = ContactMessageResource::getUrl('export');

    $this->get($index)->assertRedirect('/admin/login');
    $this->get($export)->assertRedirect('/admin/login');

    $support = inboxAdmin();
    $this->actingAs($support)->get($index)->assertOk()->assertSee('مریم')->assertSee('خروجی CSV');
    $this->actingAs($support)->get($view)->assertOk()->assertSee('یادآور برایم نمی‌آید');
    $this->actingAs($support)->get($export)->assertOk();

    $super = inboxAdmin(AdminRole::SuperAdmin);
    expect($super->can('viewAny', ContactMessage::class))->toBeTrue()
        ->and($super->can('export', ContactMessage::class))->toBeTrue()
        ->and($support->can('create', ContactMessage::class))->toBeFalse()
        ->and($support->can('delete', $this->unread))->toBeTrue();

    foreach ([AdminRole::Editor, AdminRole::SeoManager, AdminRole::ShopManager, AdminRole::DirectoryManager, null] as $role) {
        $user = inboxAdmin($role);
        $this->actingAs($user)->get($index)->assertForbidden();
        $this->actingAs($user)->get($export)->assertForbidden();
    }
});

it('shows unread, read and archived tabs and filters by topic', function (): void {
    $this->actingAs(inboxAdmin());

    Livewire::test(ListContactMessages::class)
        ->assertCanSeeTableRecords([$this->unread])
        ->assertCanNotSeeTableRecords([$this->read, $this->archived])
        ->assertSee('۱۴۰۵/۰۱/۰۱ ۰۸:۳۰')
        ->set('activeTab', 'read')
        ->assertCanSeeTableRecords([$this->read])
        ->assertCanNotSeeTableRecords([$this->unread, $this->archived])
        ->set('activeTab', 'archived')
        ->assertCanSeeTableRecords([$this->archived])
        ->set('activeTab', 'all')
        ->assertCanSeeTableRecords([$this->unread, $this->read, $this->archived])
        ->filterTable('topic', 'media')
        ->assertCanSeeTableRecords([$this->read])
        ->assertCanNotSeeTableRecords([$this->unread, $this->archived])
        ->assertActionHasUrl('export', ContactMessageResource::getUrl('export', ['topic' => 'media']));

    expect(ContactMessageResource::getNavigationBadge())->toBe('۱');
});

it('marks a message read when opened, offers a mailto reply and moves it between states', function (): void {
    $support = inboxAdmin();
    $this->actingAs($support);

    Livewire::test(ViewContactMessage::class, ['record' => $this->unread->getRouteKey()])
        ->assertActionHasUrl('reply', 'mailto:maryam@example.com?subject='.rawurlencode('پاسخ ریتمی: پشتیبانی کاربر'))
        ->callAction('archive');

    $this->unread->refresh();
    expect($this->unread->status)->toBe(ContactMessageStatus::Archived)
        ->and($this->unread->read_at)->not->toBeNull()
        ->and(ContactMessageResource::replyUrl($this->read))->toBe('tel:09123456789');

    Livewire::test(ListContactMessages::class)
        ->set('activeTab', 'read')
        ->callTableAction('markUnread', $this->read);
    $this->read->refresh();
    expect($this->read->status)->toBe(ContactMessageStatus::Unread)->and($this->read->read_at)->toBeNull();

    Livewire::test(ListContactMessages::class)
        ->callTableBulkAction('archiveBulk', [$this->read]);
    expect($this->read->refresh()->status)->toBe(ContactMessageStatus::Archived);

    $logged = Activity::query()->where('log_name', 'contact')->where('description', 'contact.status')->get();
    expect($logged)->toHaveCount(3) // opening is not logged; archive, unread, bulk archive are
        ->and($logged->first()?->causer_id)->toBe($support->id);
});

it('streams a CSV of the current tab with BOM, Persian headers, formula guard and logs the export', function (): void {
    $support = inboxAdmin();

    $response = $this->actingAs($support)->get(ContactMessageResource::getUrl('export', ['status' => 'archived']));
    $response->assertOk()->assertHeader('Content-Type', 'text/csv; charset=UTF-8')->assertDownload();

    $csv = $response->streamedContent();
    $lines = array_values(array_filter(explode("\n", substr($csv, 3))));

    expect($csv)->toStartWith("\xEF\xBB\xBF")
        ->and($lines)->toHaveCount(2)
        ->and($lines[0])->toBe('تاریخ,وضعیت,موضوع,نام,ایمیل,تلفن,پیام')
        ->and($lines[1])->toContain('"1405/01/01 08:30",بایگانی,"پشتیبانی کاربر",نگار,negar@example.com,,')
        ->and($lines[1])->toContain("'=HYPERLINK")
        ->and($csv)->not->toContain('مریم');

    $activity = Activity::query()->where('description', 'contact.exported')->sole();
    expect($activity->causer_id)->toBe($support->id)->and($activity->properties['rows'])->toBe(1);
});
