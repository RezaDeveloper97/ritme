<?php

declare(strict_types=1);

namespace App\Http\Requests;

use App\Domain\Contact\Data\ContactMessageData;
use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Enums\FormTimerResult;
use App\Domain\Contact\Support\FormTimer;
use App\Domain\Contact\Support\ReplyChannel;
use Closure;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

/**
 * The contact form (POST /contact). Spam signals — the `website` honeypot filled, or the time trap (FormTimer)
 * reporting a too-fast / missing / tampered token — switch validation off and mark the request as spam: the
 * controller then answers exactly like a successful send and stores nothing. Honest mistakes get Persian messages
 * (lang/fa/contact.php) and a redirect back to the form with the input kept.
 */
final class ContactRequest extends FormRequest
{
    public const HONEYPOT = 'website';

    public const TIMER = 'form_token';

    public const NAME_MAX = 100;

    public const MESSAGE_MIN = 10;

    public const MESSAGE_MAX = 3000;

    private ?FormTimerResult $timer = null;

    public function authorize(): bool
    {
        return true;
    }

    public function isSpam(): bool
    {
        return trim((string) $this->input(self::HONEYPOT, '')) !== '' || $this->timer()->isSpam();
    }

    /**
     * @return array<string, mixed>
     */
    public function rules(): array
    {
        if ($this->isSpam()) {
            return [];
        }

        return [
            'topic' => ['required', 'string', Rule::enum(ContactTopic::class)],
            'name' => ['required', 'string', 'max:'.self::NAME_MAX],
            'contact' => ['required', 'string', 'max:191', static function (string $attribute, mixed $value, Closure $fail): void {
                if (ReplyChannel::parse(is_string($value) ? $value : null) === null) {
                    $fail(__('contact.validation.contact_format'));
                }
            }],
            'message' => ['required', 'string', 'min:'.self::MESSAGE_MIN, 'max:'.self::MESSAGE_MAX],
            self::TIMER => [function (string $attribute, mixed $value, Closure $fail): void {
                if ($this->timer() === FormTimerResult::Expired) {
                    $fail(__('contact.validation.expired'));
                }
            }],
        ];
    }

    /**
     * @return array<string, string>
     */
    public function messages(): array
    {
        $messages = [];
        foreach (['topic.required', 'topic.enum', 'name.required', 'name.max', 'contact.required', 'contact.max', 'message.required', 'message.min', 'message.max'] as $key) {
            $messages[$key] = __('contact.validation.'.str_replace('.', '_', $key), [
                'name_max' => fa_digits(self::NAME_MAX),
                'min' => fa_digits(self::MESSAGE_MIN),
                'max' => fa_digits(self::MESSAGE_MAX),
            ]);
        }
        $messages['topic.string'] = $messages['topic.enum'];
        $messages['name.string'] = $messages['name.required'];
        $messages['contact.string'] = $messages['contact.required'];
        $messages['message.string'] = $messages['message.required'];

        return $messages;
    }

    public function toData(): ContactMessageData
    {
        $channel = ReplyChannel::parse($this->string('contact')->toString());
        assert($channel !== null);

        return new ContactMessageData(
            topic: ContactTopic::from($this->string('topic')->toString()),
            name: $this->string('name')->squish()->toString(),
            channel: $channel,
            message: trim(str_replace("\r\n", "\n", $this->string('message')->toString())),
        );
    }

    protected function getRedirectUrl(): string
    {
        return route('contact').'#contact-form';
    }

    private function timer(): FormTimerResult
    {
        return $this->timer ??= $this->container->make(FormTimer::class)->check(
            is_string($token = $this->input(self::TIMER)) ? $token : null,
        );
    }
}
