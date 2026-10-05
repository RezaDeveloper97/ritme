<?php

declare(strict_types=1);

namespace App\Http\Requests;

use App\Domain\Contact\Enums\FormTimerResult;
use App\Domain\Contact\Support\FormTimer;
use App\Domain\Contact\Support\ReplyChannel;
use App\Domain\Shop\Ordering\Data\CheckoutSubmission;
use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use App\Domain\Shop\Ordering\Support\DeliverySlots;
use App\Domain\Shop\Ordering\Support\IranProvinces;
use App\Domain\Shop\Payment\Contracts\PaymentGateway;
use App\Support\Text\PersianDigits;
use Carbon\CarbonImmutable;
use Closure;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

/**
 * The checkout form (POST /shop/checkout, L6-05). Only the recipient's minimal data and delivery preferences — no
 * price, quantity or product ever comes from here (PlaceOrder re-reads the live cart).
 *
 * Anti-spam like the contact / booking forms (no captcha): the `website` honeypot, or a FormTimer token that is
 * missing, tampered or came back too fast, marks the post as spam (validation skipped; the controller stores nothing
 * and asks to try again). An expired token is a normal «send again» validation error.
 *
 * Normalisation: Persian/Arabic digits → Latin; the mobile becomes `09xxxxxxxxx` (+98 / 0098 accepted); the postal
 * code is 10 digits (spaces/dashes dropped); the delivery slot is `Y-m-d|window` from the slot radios.
 */
final class CheckoutRequest extends FormRequest
{
    public const HONEYPOT = 'website';

    public const TIMER = 'form_token';

    public const NAME_MAX = 100;

    public const CITY_MAX = 60;

    public const ADDRESS_MAX = 500;

    public const NOTE_MAX = 500;

    private ?FormTimerResult $timer = null;

    public function authorize(): bool
    {
        return true;
    }

    public function isSpam(): bool
    {
        return trim((string) $this->input(self::HONEYPOT, '')) !== '' || $this->timer()->isSpam();
    }

    protected function prepareForValidation(): void
    {
        $latin = static fn (mixed $value): mixed => is_string($value) ? trim(PersianDigits::toLatin($value)) : $value;
        $postal = $latin($this->input('postal_code'));

        $this->merge(array_filter([
            'mobile' => $latin($this->input('mobile')),
            'postal_code' => is_string($postal) ? (preg_replace('/[\s\-]/u', '', $postal) ?? '') : $postal,
        ], static fn (mixed $value): bool => $value !== null));
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
            'name' => ['required', 'string', 'min:2', 'max:'.self::NAME_MAX],
            'mobile' => ['required', 'string', 'max:30', static function (string $attribute, mixed $value, Closure $fail): void {
                if (ReplyChannel::parse(is_string($value) && ! str_contains($value, '@') ? $value : null)?->phone === null) {
                    $fail(__('shop.checkout.validation.mobile'));
                }
            }],
            'province' => ['required', 'string', static function (string $attribute, mixed $value, Closure $fail): void {
                if (! is_string($value) || ! IranProvinces::has($value)) {
                    $fail(__('shop.checkout.validation.province'));
                }
            }],
            'city' => ['required', 'string', 'min:2', 'max:'.self::CITY_MAX],
            'address' => ['required', 'string', 'min:10', 'max:'.self::ADDRESS_MAX],
            'postal_code' => ['nullable', 'string', 'regex:/^\d{10}$/'],
            'note' => ['nullable', 'string', 'max:'.self::NOTE_MAX],
            'delivery' => ['required', 'string', 'max:40', static function (string $attribute, mixed $value, Closure $fail): void {
                if (self::slot(is_string($value) ? $value : '') === null) {
                    $fail(__('shop.checkout.validation.delivery'));
                }
            }],
            'payment' => ['required', Rule::in([$this->container->make(PaymentGateway::class)->method()->value])],
            'checkout_token' => ['required', 'string', 'size:40'],
            'cart_signature' => ['required', 'string', 'max:64'],
            self::TIMER => [function (string $attribute, mixed $value, Closure $fail): void {
                if ($this->timer() === FormTimerResult::Expired) {
                    $fail(__('shop.checkout.validation.expired'));
                }
            }],
        ];
    }

    /**
     * One Persian message per field (lang/fa/shop.php `checkout.validation`), whatever rule failed.
     *
     * @return array<string, string>
     */
    public function messages(): array
    {
        $params = [
            'name_max' => fa_digits(self::NAME_MAX), 'address_max' => fa_digits(self::ADDRESS_MAX),
            'note_max' => fa_digits(self::NOTE_MAX), 'city_max' => fa_digits(self::CITY_MAX),
        ];
        $messages = [];
        foreach (['name', 'mobile', 'province', 'city', 'address', 'postal_code', 'note', 'delivery', 'payment', 'checkout_token', 'cart_signature'] as $field) {
            $messages[$field] = __('shop.checkout.validation.'.$field, $params);
        }

        return $messages;
    }

    public function toSubmission(): CheckoutSubmission
    {
        $mobile = ReplyChannel::parse($this->string('mobile')->toString())?->phone;
        $slot = self::slot($this->string('delivery')->toString());
        assert($mobile !== null && $slot !== null);

        return new CheckoutSubmission(
            recipientName: $this->string('name')->squish()->toString(),
            mobile: $mobile,
            province: $this->string('province')->toString(),
            city: $this->string('city')->squish()->toString(),
            address: $this->string('address')->squish()->toString(),
            postalCode: $this->filled('postal_code') ? $this->string('postal_code')->toString() : null,
            note: self::optional(trim(str_replace("\r\n", "\n", $this->string('note')->toString()))),
            deliveryDate: $slot[0],
            deliveryWindow: $slot[1],
            discreetPackaging: $this->boolean('discreet'),
            token: $this->string('checkout_token')->toString(),
            signature: $this->string('cart_signature')->toString(),
        );
    }

    /**
     * `2026-10-08|morning` → [day, window] when the day is still acceptable, else null.
     *
     * @return array{0: CarbonImmutable, 1: DeliveryWindow}|null
     */
    public static function slot(string $value): ?array
    {
        [$date, $window] = array_pad(explode('|', $value, 2), 2, '');
        $day = DeliverySlots::parse($date);
        $window = DeliveryWindow::tryFrom($window);

        return $day === null || $window === null || ! DeliverySlots::isAcceptable($day) ? null : [$day, $window];
    }

    protected function getRedirectUrl(): string
    {
        return route('shop.checkout').'#checkout-form';
    }

    private static function optional(string $value): ?string
    {
        return $value === '' ? null : $value;
    }

    private function timer(): FormTimerResult
    {
        return $this->timer ??= $this->container->make(FormTimer::class)->check(
            is_string($token = $this->input(self::TIMER)) ? $token : null,
        );
    }
}
