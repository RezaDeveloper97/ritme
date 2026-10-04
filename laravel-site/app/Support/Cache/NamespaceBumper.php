<?php

declare(strict_types=1);

namespace App\Support\Cache;

use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Database\Eloquent\Model;
use RuntimeException;

/**
 * Bumps a model's namespaces plus cacheaside.always_bump (`pages`) after the surrounding transaction
 * commits, so no reader can re-cache pre-commit data under the new version. Used by
 * CacheBumpingObserver; inject it wherever else a model change must invalidate caches.
 */
final class NamespaceBumper
{
    public function __construct(private readonly NamespaceVersions $versions, private readonly Config $config) {}

    /**
     * @param  list<string>  $namespaces
     */
    public function bumpFor(Model $model, array $namespaces): void
    {
        $all = array_values(array_unique([...$namespaces, ...$this->alwaysBump()]));

        if ($all === []) {
            return;
        }

        $bump = fn (): array => $this->versions->bump(...$all);

        try {
            $model->getConnection()->afterCommit($bump);
        } catch (RuntimeException) {
            $bump(); // connection without a transactions manager
        }
    }

    /**
     * @return list<string>
     */
    private function alwaysBump(): array
    {
        return array_values(array_map(strval(...), (array) $this->config->get('cacheaside.always_bump', [])));
    }
}
