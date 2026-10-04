<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Places\Pages;

use App\Domain\Directory\Actions\SavePlace;
use App\Domain\Directory\Models\Place;
use App\Filament\Resources\Directory\DirectoryAdmin;
use App\Filament\Resources\Directory\Places\PlaceFormData;
use App\Filament\Resources\Directory\Places\PlaceResource;
use Filament\Resources\Pages\CreateRecord;
use Illuminate\Database\Eloquent\Model;

/**
 * A new place starts as a draft (publish it from the edit page once it is complete).
 */
final class CreatePlace extends CreateRecord
{
    protected static string $resource = PlaceResource::class;

    protected function handleRecordCreation(array $data): Model
    {
        return app(SavePlace::class)->handle(
            new Place,
            PlaceFormData::attributes($data),
            PlaceFormData::amenityIds($data),
            PlaceFormData::galleryIds($data),
            DirectoryAdmin::user(),
        );
    }
}
