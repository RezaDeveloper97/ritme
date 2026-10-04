<?php

declare(strict_types=1);

namespace App\Domain\Contact\Models;

use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Enums\ContactTopic;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * A message from the public contact form (admin inbox). Exactly one of email / phone is set — the channel the
 * visitor asked to be answered on.
 *
 * @property int $id
 * @property ContactTopic $topic
 * @property string $name
 * @property string|null $email
 * @property string|null $phone
 * @property string $message
 * @property ContactMessageStatus $status
 * @property Carbon|null $read_at
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 */
final class ContactMessage extends Model
{
    protected $table = 'contact_messages';

    protected $fillable = ['topic', 'name', 'email', 'phone', 'message', 'status', 'read_at'];

    protected $attributes = [
        'status' => 'unread',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'topic' => ContactTopic::class,
            'status' => ContactMessageStatus::class,
            'read_at' => 'datetime',
        ];
    }

    /** The reply channel as typed back to the admin: the email, else the phone number. */
    public function replyTo(): string
    {
        return $this->email ?? $this->phone ?? '';
    }
}
