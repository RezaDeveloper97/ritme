<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Actions;

use App\Domain\Directory\Join\Enums\JoinRequestStatus;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Directory\Support\DirectoryActivity;
use Illuminate\Database\Eloquent\Model;
use InvalidArgumentException;

/**
 * Rejects a pending join request (admin, L5-06) and writes the change to the activity log. The photos stay in the
 * media library until «delete unused» (they are registered as usages while the request exists).
 */
final class RejectJoinRequest
{
    /**
     * @throws InvalidArgumentException when the request is not pending
     */
    public function handle(JoinRequest $request, ?Model $causer = null): void
    {
        if ($request->status !== JoinRequestStatus::Pending) {
            throw new InvalidArgumentException('Only pending join requests can be rejected.');
        }

        $request->status = JoinRequestStatus::Rejected;
        $request->save();

        DirectoryActivity::status($request, 'directory.join.rejected', JoinRequestStatus::Pending->value, JoinRequestStatus::Rejected->value, $causer);
    }
}
