<?php

declare(strict_types=1);

namespace App\Http\Requests;

use App\Domain\Contact\Enums\FormTimerResult;
use App\Domain\Contact\Support\FormTimer;
use App\Domain\Contact\Support\ReplyChannel;
use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Data\AmenityData;
use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\DistrictData;
use App\Domain\Directory\Enums\Weekday;
use App\Domain\Directory\Join\Data\JoinRequestData;
use App\Domain\Directory\Join\Enums\AgeGroup;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Join\Support\JoinForm;
use App\Domain\Media\Data\MediaUpload;
use App\Support\Text\PersianDigits;
use Closure;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Http\UploadedFile;
use Illuminate\Validation\Rule;

/**
 * The join form (POST /directory/join) — one long form that the `stepper` module splits into four steps; the server
 * validates every field on every post. Anti-spam like the contact form (no captcha): the `website` honeypot or a
 * FormTimer time trap that reports a too-fast / missing / tampered token switches validation off and marks the
 * request as spam — the controller then answers like a real submit and stores nothing. Photos are only checked here
 * (count, type, size); SubmitJoinRequest sends them through the media pipeline. Copy: lang/fa/directory.php `join`.
 */
final class JoinRequest extends FormRequest
{
    public const HONEYPOT = 'website';

    public const TIMER = 'form_token';

    /** Field (dot path, first segment) => form step, so a failed post reopens the first step with an error. */
    public const STEP_OF = [
        'name' => 1, 'category' => 1, 'city' => 1, 'district' => 1, 'contact_name' => 1, 'mobile' => 1, 'email' => 1,
        'address' => 2, 'phone' => 2, 'latitude' => 2, 'longitude' => 2, 'photos' => 2, 'about' => 2, 'ages' => 2,
        'amenities' => 2, 'hours' => 2,
        'services' => 3, 'booking_mode' => 3,
        'license' => 4, 'terms' => 4, self::TIMER => 4,
    ];

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

        $hours = $this->input('hours');
        if (is_array($hours)) {
            foreach ($hours as $day => $row) {
                if (is_array($row)) {
                    $hours[$day] = array_map($latin, $row);
                }
            }
        }

        $this->merge(array_filter([
            'latitude' => $latin($this->input('latitude')),
            'longitude' => $latin($this->input('longitude')),
            'phone' => $latin($this->input('phone')),
            'hours' => $hours,
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

        $taxonomy = $this->taxonomy();
        $cities = $taxonomy->cities();

        return [
            'name' => ['required', 'string', 'min:2', 'max:'.JoinForm::NAME_MAX],
            'category' => ['required', 'integer', Rule::in(array_map(static fn (CategoryData $c): int => $c->id, $taxonomy->categories()))],
            'city' => ['required', 'integer', Rule::in(array_map(static fn (CityData $c): int => $c->id, $cities))],
            'district' => ['nullable', 'integer', function (string $attribute, mixed $value, Closure $fail) use ($cities): void {
                if (! in_array((int) $value, self::districtIds($cities, (int) $this->input('city')), true)) {
                    $fail(__('directory.join.validation.district'));
                }
            }],
            'contact_name' => ['required', 'string', 'min:2', 'max:100'],
            'mobile' => ['required', 'string', 'max:30', static function (string $attribute, mixed $value, Closure $fail): void {
                if (ReplyChannel::parse(is_string($value) && ! str_contains($value, '@') ? $value : null)?->phone === null) {
                    $fail(__('directory.join.validation.mobile'));
                }
            }],
            'email' => ['nullable', 'string', 'max:191', 'email:rfc'],

            'address' => ['required', 'string', 'min:10', 'max:500'],
            'phone' => ['nullable', 'string', 'regex:/^[0-9+\-\s()]{5,20}$/'],
            'latitude' => ['nullable', 'numeric', 'between:24,40', 'required_with:longitude'],
            'longitude' => ['nullable', 'numeric', 'between:43,64', 'required_with:latitude'],
            'photos' => ['nullable', 'array', 'max:'.JoinForm::PHOTOS_MAX],
            'photos.*' => ['file', 'mimetypes:'.implode(',', JoinForm::PHOTO_MIMES), 'max:'.intdiv(JoinForm::photoMaxBytes(), 1024)],
            'about' => ['nullable', 'string', 'max:'.JoinForm::ABOUT_MAX],
            'ages' => ['nullable', 'array'],
            'ages.*' => ['string', Rule::enum(AgeGroup::class)],
            'amenities' => ['nullable', 'array'],
            'amenities.*' => ['integer', Rule::in(array_map(static fn (AmenityData $a): int => $a->id, $taxonomy->amenities()))],
            'hours' => ['nullable', 'array:'.implode(',', array_map(static fn (Weekday $d): string => $d->value, Weekday::cases()))],
            'hours.*' => ['array'],
            'hours.*.open' => ['nullable', 'boolean'],
            'hours.*.opens' => ['nullable', 'required_if_accepted:hours.*.open', 'date_format:H:i'],
            'hours.*.closes' => ['nullable', 'required_if_accepted:hours.*.open', 'date_format:H:i'],

            'services' => ['nullable', 'string', 'max:'.JoinForm::SERVICES_MAX],
            'booking_mode' => ['required', Rule::enum(BookingMode::class)],

            'license' => ['accepted'],
            'terms' => ['accepted'],
            self::TIMER => [function (string $attribute, mixed $value, Closure $fail): void {
                if ($this->timer() === FormTimerResult::Expired) {
                    $fail(__('directory.join.validation.expired'));
                }
            }],
        ];
    }

    /**
     * One Persian message per field (lang/fa/directory.php `join.validation`), whatever rule failed.
     *
     * @return array<string, string>
     */
    public function messages(): array
    {
        $params = [
            'name_max' => fa_digits(JoinForm::NAME_MAX),
            'about_max' => fa_digits(JoinForm::ABOUT_MAX),
            'services_max' => fa_digits(JoinForm::SERVICES_MAX),
            'photos_max' => fa_digits(JoinForm::PHOTOS_MAX),
            'photo_mb' => fa_digits(round(JoinForm::photoMaxBytes() / 1048576, 1)),
        ];

        $messages = [];
        foreach ([
            'name', 'category', 'city', 'contact_name', 'mobile', 'email', 'address', 'phone', 'latitude', 'longitude',
            'photos', 'about', 'ages', 'amenities', 'services', 'booking_mode', 'license', 'terms',
        ] as $field) {
            $messages[$field] = __('directory.join.validation.'.$field, $params);
        }
        $messages['district.integer'] = __('directory.join.validation.district');
        $messages['photos.*'] = __('directory.join.validation.photo', $params);
        $messages['ages.*'] = $messages['ages'];
        $messages['amenities.*'] = $messages['amenities'];
        $messages['hours'] = __('directory.join.validation.hours');
        $messages['hours.*'] = __('directory.join.validation.hours');

        return $messages;
    }

    public function toData(): JoinRequestData
    {
        $mobile = ReplyChannel::parse($this->string('mobile')->toString())?->phone;
        assert($mobile !== null);
        $name = $this->string('name')->squish()->toString();

        $photos = [];
        foreach ((array) $this->file('photos', []) as $file) {
            if ($file instanceof UploadedFile && $file->isValid()) {
                $photos[] = new MediaUpload(
                    path: (string) $file->getRealPath(),
                    originalName: $file->getClientOriginalName(),
                    alt: __('directory.join.photo_alt', ['name' => $name]),
                );
            }
        }

        $district = $this->input('district');

        return new JoinRequestData(
            name: $name,
            categoryId: $this->integer('category'),
            cityId: $this->integer('city'),
            districtId: $district === null || $district === '' ? null : (int) $district,
            contactName: $this->string('contact_name')->squish()->toString(),
            mobile: $mobile,
            email: self::optional(mb_strtolower($this->string('email')->trim()->toString())),
            address: $this->string('address')->squish()->toString(),
            phone: self::optional($this->string('phone')->squish()->toString()),
            latitude: $this->filled('latitude') ? round((float) $this->input('latitude'), 7) : null,
            longitude: $this->filled('longitude') ? round((float) $this->input('longitude'), 7) : null,
            about: self::optional(trim(str_replace("\r\n", "\n", $this->string('about')->toString()))),
            ageGroups: array_values(array_unique(array_map(
                static fn (mixed $value): AgeGroup => AgeGroup::from((string) $value),
                (array) $this->input('ages', []),
            ), SORT_REGULAR)),
            amenityIds: array_values(array_unique(array_map(intval(...), (array) $this->input('amenities', [])))),
            openingHours: $this->openingHours(),
            services: self::optional(trim(str_replace("\r\n", "\n", $this->string('services')->toString()))),
            bookingMode: BookingMode::from($this->string('booking_mode')->toString()),
            photos: $photos,
        );
    }

    /**
     * The hours rows as the directory_places JSON: open days get one range, unticked days are closed (`[]`).
     *
     * @return array<string, list<array{opens: string, closes: string}>>
     */
    private function openingHours(): array
    {
        $rows = (array) $this->input('hours', []);
        $hours = [];
        foreach (Weekday::cases() as $day) {
            $row = (array) ($rows[$day->value] ?? []);
            $hours[$day->value] = filter_var($row['open'] ?? false, FILTER_VALIDATE_BOOLEAN)
                ? [['opens' => (string) $row['opens'], 'closes' => (string) $row['closes']]]
                : [];
        }

        return $hours;
    }

    protected function getRedirectUrl(): string
    {
        return route('directory.join').'#join-form';
    }

    /**
     * @param  list<CityData>  $cities
     * @return list<int>
     */
    private static function districtIds(array $cities, int $cityId): array
    {
        foreach ($cities as $city) {
            if ($city->id === $cityId) {
                return array_map(static fn (DistrictData $d): int => $d->id, $city->districts);
            }
        }

        return [];
    }

    private static function optional(string $value): ?string
    {
        return $value === '' ? null : $value;
    }

    private function taxonomy(): TaxonomyRepository
    {
        return $this->container->make(TaxonomyRepository::class);
    }

    private function timer(): FormTimerResult
    {
        return $this->timer ??= $this->container->make(FormTimer::class)->check(
            is_string($token = $this->input(self::TIMER)) ? $token : null,
        );
    }
}
