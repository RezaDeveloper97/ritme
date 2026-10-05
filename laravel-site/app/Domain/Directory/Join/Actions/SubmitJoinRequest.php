<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Actions;

use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Support\ContactRecipients;
use App\Domain\Directory\Join\Data\JoinRequestData;
use App\Domain\Directory\Join\Enums\AgeGroup;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Exceptions\InvalidMediaException;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Notifications\JoinRequestReceived;
use Illuminate\Contracts\Notifications\Dispatcher;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Notifications\AnonymousNotifiable;
use Psr\Log\LoggerInterface;
use Throwable;

/**
 * Stores a join request as pending with its photos: every photo goes through the media pipeline first (StoreMedia —
 * sniffed type, size/pixel limits, metadata stripped) onto the private PHOTO_DISK (L9-04b, F17: nothing a visitor
 * uploads is web-reachable before an admin approves it; ApproveJoinRequest promotes the photos to the media disk and
 * admins preview them through the panel's pending-media route), then the request and its ordered photo list are
 * written in one transaction with a fresh public tracking code. If a photo is refused or the write fails, the photos
 * this submission created are deleted again. The partnership mailbox (settings,
 * support as fallback) gets a data-minimal mail; mail problems are logged, never shown — the request is already safe.
 *
 * @throws InvalidMediaException when a photo is refused by the pipeline (nothing is stored for the request then)
 */
final class SubmitJoinRequest
{
    private const CODE_ATTEMPTS = 10;

    /** Private disk (config/filesystems.php) that holds join photos until approval. */
    public const PHOTO_DISK = 'pending';

    public function __construct(
        private readonly StoreMedia $storeMedia,
        private readonly ConnectionInterface $db,
        private readonly SettingsRepository $settings,
        private readonly Dispatcher $notifications,
        private readonly LoggerInterface $log,
    ) {}

    public function handle(JoinRequestData $data): JoinRequest
    {
        $mediaIds = [];
        $created = [];

        try {
            foreach ($data->photos as $photo) {
                $media = $this->storeMedia->handle(self::pending($photo));
                $mediaIds[] = $media->id;
                if ($media->wasRecentlyCreated) {
                    $created[] = $media;
                }
            }

            $request = $this->store($data, $mediaIds);
        } catch (Throwable $e) {
            foreach ($created as $media) {
                $media->delete(); // MediaObserver removes the files
            }

            throw $e;
        }

        $this->notify($request);

        return $request;
    }

    private static function pending(MediaUpload $photo): MediaUpload
    {
        return new MediaUpload(
            path: $photo->path,
            originalName: $photo->originalName,
            alt: $photo->alt,
            title: $photo->title,
            caption: $photo->caption,
            focalX: $photo->focalX,
            focalY: $photo->focalY,
            uploadedBy: $photo->uploadedBy,
            disk: self::PHOTO_DISK,
        );
    }

    /**
     * @param  list<int>  $mediaIds
     */
    private function store(JoinRequestData $data, array $mediaIds): JoinRequest
    {
        /** @var JoinRequest $request */
        $request = $this->db->transaction(function () use ($data, $mediaIds): JoinRequest {
            $request = JoinRequest::query()->create([
                'code' => $this->uniqueCode(),
                'name' => $data->name,
                'category_id' => $data->categoryId,
                'city_id' => $data->cityId,
                'district_id' => $data->districtId,
                'contact_name' => $data->contactName,
                'mobile' => $data->mobile,
                'email' => $data->email,
                'address' => $data->address,
                'phone' => $data->phone,
                'latitude' => $data->latitude,
                'longitude' => $data->longitude,
                'about' => $data->about,
                'age_groups' => array_map(static fn (AgeGroup $group): string => $group->value, $data->ageGroups),
                'amenity_ids' => $data->amenityIds,
                'opening_hours' => $data->openingHours,
                'services' => $data->services,
                'booking_mode' => $data->bookingMode,
                'terms_accepted_at' => now(),
            ]);

            $ordered = [];
            foreach (array_values(array_unique($mediaIds)) as $position => $mediaId) {
                $ordered[$mediaId] = ['sort_order' => $position];
            }
            $request->photos()->sync($ordered);

            return $request;
        });

        return $request;
    }

    /** A 6-digit code (no leading zero) that no other request uses. */
    private function uniqueCode(): string
    {
        for ($i = 0; $i < self::CODE_ATTEMPTS; $i++) {
            $code = (string) random_int(100000, 999999);
            if (! JoinRequest::query()->where('code', $code)->exists()) {
                return $code;
            }
        }

        return (string) random_int(10000000, 99999999);
    }

    private function notify(JoinRequest $request): void
    {
        $recipient = ContactRecipients::for(ContactTopic::Partnership, $this->settings->all());
        if ($recipient === null) {
            $this->log->warning('Join request stored without notification: no valid partnership/support email in settings.', ['id' => $request->id]);

            return;
        }

        try {
            $this->notifications->send((new AnonymousNotifiable)->route('mail', $recipient), new JoinRequestReceived($request));
        } catch (Throwable $e) {
            report($e);
        }
    }
}
