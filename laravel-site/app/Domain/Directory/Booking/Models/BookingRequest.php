<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Models;

use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Enums\TimeWindow;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceService;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;

/**
 * A booking request from the place page (L5-04), written only by CreateBookingRequest and handled in the admin
 * (L5-06). `place_name` / `service_*` are snapshots taken when the request was sent.
 *
 * @property int $id
 * @property string $code
 * @property BookingStatus $status
 * @property int|null $place_id
 * @property string $place_name
 * @property int|null $service_id
 * @property string|null $service_name
 * @property int|null $service_price
 * @property string|null $service_price_unit
 * @property Carbon $preferred_date
 * @property TimeWindow $time_window
 * @property string $parent_name
 * @property string $mobile
 * @property int|null $child_age_months
 * @property string|null $note
 * @property Carbon|null $created_at
 * @property Carbon|null $updated_at
 * @property-read Place|null $place
 * @property-read PlaceService|null $service
 */
final class BookingRequest extends Model
{
    protected $table = 'directory_booking_requests';

    protected $fillable = [
        'code', 'status', 'place_id', 'place_name', 'service_id', 'service_name', 'service_price',
        'service_price_unit', 'preferred_date', 'time_window', 'parent_name', 'mobile', 'child_age_months', 'note',
    ];

    protected $attributes = [
        'status' => 'new',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'status' => BookingStatus::class,
            'time_window' => TimeWindow::class,
            'preferred_date' => 'date',
            'service_price' => 'integer',
            'child_age_months' => 'integer',
        ];
    }

    /**
     * @return BelongsTo<Place, $this>
     */
    public function place(): BelongsTo
    {
        return $this->belongsTo(Place::class, 'place_id');
    }

    /**
     * @return BelongsTo<PlaceService, $this>
     */
    public function service(): BelongsTo
    {
        return $this->belongsTo(PlaceService::class, 'service_id');
    }
}
