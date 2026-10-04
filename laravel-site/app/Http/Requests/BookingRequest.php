<?php

declare(strict_types=1);

namespace App\Http\Requests;

use App\Domain\Contact\Enums\FormTimerResult;
use App\Domain\Contact\Support\FormTimer;
use App\Domain\Contact\Support\ReplyChannel;
use App\Domain\Directory\Booking\Data\BookingSubmission;
use App\Domain\Directory\Booking\Enums\TimeWindow;
use App\Domain\Directory\Booking\Support\BookingDays;
use App\Domain\Directory\Booking\Support\ChildAge;
use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Data\PlaceServiceData;
use App\Support\Jalali\JalaliCalendar;
use App\Support\Text\PersianDigits;
use Carbon\CarbonImmutable;
use Closure;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;
use Throwable;

/**
 * The booking request form of the place page (POST /directory/place/{slug}/book, L5-04). Errors go to the `booking`
 * error bag (the review form on the same page has its own fields) and back to `#book`.
 *
 * Anti-spam like the contact / join forms (no captcha): the `homepage` honeypot, or a FormTimer token that is missing,
 * tampered or came back too fast, marks the post as spam — validation is skipped and the controller answers exactly
 * like a real submit, storing nothing. The place page is full-page cached, so its token carries the time the page
 * was rendered into the cache, which is never later than the moment the visitor saw it: the "too fast" check can
 * only get more lenient on a cache HIT, never reject a person. `throttle:directory-booking` limits posts per IP and
 * per mobile (DirectoryServiceProvider).
 *
 * Normalisation: Persian/Arabic digits → Latin; the mobile becomes `09xxxxxxxxx` (+98 / 0098 accepted); the day is
 * an ISO date from the day chips or a typed Jalali date (`۱۴۰۵/۰۷/۱۶`).
 */
final class BookingRequest extends FormRequest
{
    public const HONEYPOT = 'homepage';

    public const TIMER = 'form_token';

    public const NAME_MAX = 100;

    public const NOTE_MAX = 500;

    /** @var string */
    protected $errorBag = 'booking';

    private ?FormTimerResult $timer = null;

    private ?PlaceData $place = null;

    public function authorize(): bool
    {
        return true;
    }

    public function isSpam(): bool
    {
        return trim((string) $this->input(self::HONEYPOT, '')) !== '' || $this->timer()->isSpam();
    }

    /** The published place of the route (404 when it is gone). */
    public function place(): PlaceData
    {
        if ($this->place === null) {
            $slug = $this->route('slug');
            $place = is_string($slug) ? $this->container->make(PlaceRepository::class)->findPublishedBySlug($slug) : null;
            if ($place === null) {
                abort(404);
            }
            $this->place = $place;
        }

        return $this->place;
    }

    protected function prepareForValidation(): void
    {
        $latin = static fn (mixed $value): mixed => is_string($value) ? trim(PersianDigits::toLatin($value)) : $value;

        $this->merge(array_filter([
            'service' => $latin($this->input('service')),
            'date' => $latin($this->input('date')),
            'child_age' => $latin($this->input('child_age')),
        ], static fn (mixed $value): bool => $value !== null));
    }

    /**
     * @return array<string, mixed>
     */
    public function rules(): array
    {
        $place = $this->place();
        if ($this->isSpam()) {
            return [];
        }

        $serviceIds = array_map(static fn (PlaceServiceData $s): int => $s->id, $place->services);

        return [
            'service' => $serviceIds === [] ? ['nullable'] : ['required', 'integer', Rule::in($serviceIds)],
            'date' => ['required', 'string', 'max:20', static function (string $attribute, mixed $value, Closure $fail) use ($place): void {
                $date = self::parseDate(is_string($value) ? $value : '');
                if ($date === null || ! BookingDays::isBookable($date, $place->openingHours())) {
                    $fail(__('directory.booking.validation.date'));
                }
            }],
            'time_window' => ['required', Rule::enum(TimeWindow::class)],
            'name' => ['required', 'string', 'min:2', 'max:'.self::NAME_MAX],
            'mobile' => ['required', 'string', 'max:30', static function (string $attribute, mixed $value, Closure $fail): void {
                if (ReplyChannel::parse(is_string($value) && ! str_contains($value, '@') ? $value : null)?->phone === null) {
                    $fail(__('directory.booking.validation.mobile'));
                }
            }],
            'child_age' => ['nullable', 'integer', 'min:0', 'max:'.ChildAge::MAX_MONTHS, function (string $attribute, mixed $value, Closure $fail): void {
                if (ChildAge::toMonths((int) $value, $this->string('child_age_unit')->toString()) > ChildAge::MAX_MONTHS) {
                    $fail(__('directory.booking.validation.child_age'));
                }
            }],
            'child_age_unit' => ['nullable', Rule::in(['month', 'year'])],
            'note' => ['nullable', 'string', 'max:'.self::NOTE_MAX],
            self::TIMER => [function (string $attribute, mixed $value, Closure $fail): void {
                if ($this->timer() === FormTimerResult::Expired) {
                    $fail(__('directory.booking.validation.expired'));
                }
            }],
        ];
    }

    /**
     * One Persian message per field (lang/fa/directory.php `booking.validation`), whatever rule failed.
     *
     * @return array<string, string>
     */
    public function messages(): array
    {
        $params = ['name_max' => fa_digits(self::NAME_MAX), 'note_max' => fa_digits(self::NOTE_MAX)];
        $messages = [];
        foreach (['service', 'date', 'time_window', 'name', 'mobile', 'child_age', 'child_age_unit', 'note'] as $field) {
            $messages[$field] = __('directory.booking.validation.'.$field, $params);
        }

        return $messages;
    }

    public function toSubmission(): BookingSubmission
    {
        $mobile = ReplyChannel::parse($this->string('mobile')->toString())?->phone;
        $date = self::parseDate($this->string('date')->toString());
        assert($mobile !== null && $date !== null);

        return new BookingSubmission(
            serviceId: $this->filled('service') ? $this->integer('service') : null,
            date: $date,
            window: TimeWindow::from($this->string('time_window')->toString()),
            parentName: $this->string('name')->squish()->toString(),
            mobile: $mobile,
            childAgeMonths: $this->filled('child_age') ? ChildAge::toMonths($this->integer('child_age'), $this->string('child_age_unit')->toString()) : null,
            note: self::optional(trim(str_replace("\r\n", "\n", $this->string('note')->toString()))),
        );
    }

    /**
     * A spam post is answered with the booked page of what it sent, but nothing is stored — this is that "would-be"
     * request built leniently (no validation ran): unknown values fall back to harmless defaults.
     */
    public function toSpamSubmission(): BookingSubmission
    {
        $window = TimeWindow::tryFrom($this->string('time_window')->toString()) ?? TimeWindow::Any;
        $age = $this->input('child_age');

        return new BookingSubmission(
            serviceId: is_numeric($this->input('service')) ? (int) $this->input('service') : null,
            date: self::parseDate($this->string('date')->toString()) ?? BookingDays::today()->addDay(),
            window: $window,
            parentName: $this->string('name')->squish()->limit(self::NAME_MAX, '')->toString(),
            mobile: ReplyChannel::parse($this->string('mobile')->toString())->phone ?? '',
            childAgeMonths: is_numeric($age) && (int) $age >= 0 ? min(ChildAge::MAX_MONTHS, ChildAge::toMonths((int) $age, $this->string('child_age_unit')->toString())) : null,
        );
    }

    /**
     * `2026-10-08` (the day chips) or a Jalali `1405/07/16` / `1405-7-16` (typed), as a Tehran calendar day.
     */
    public static function parseDate(string $value): ?CarbonImmutable
    {
        $value = trim(PersianDigits::toLatin($value));
        if (preg_match('~^(\d{4})[-/](\d{1,2})[-/](\d{1,2})$~', $value, $m) !== 1) {
            return null;
        }
        [$y, $mo, $d] = [(int) $m[1], (int) $m[2], (int) $m[3]];

        try {
            if ($y >= 1300 && $y < 1500) {
                if (! JalaliCalendar::isValid($y, $mo, $d)) {
                    return null;
                }
                [$y, $mo, $d] = JalaliCalendar::toGregorian($y, $mo, $d);
            } elseif (! checkdate($mo, $d, $y)) {
                return null;
            }

            return CarbonImmutable::create($y, $mo, $d, 0, 0, 0, BookingDays::TIMEZONE);
        } catch (Throwable) {
            return null;
        }
    }

    protected function getRedirectUrl(): string
    {
        return route('directory.place', [$this->place()->slug]).'#book';
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
