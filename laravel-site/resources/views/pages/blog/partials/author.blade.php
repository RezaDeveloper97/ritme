{{--
    Author / medical reviewer profile inside the page intro (no design page: built from the kit's tokens). $author =
    AuthorData; avatar through <x-picture> (alt = name), credentials and sameAs profile links (rel="me noopener").
--}}
<div class="flex max-w-205 items-start gap-5 max-sm:flex-col">
    @if ($author->avatarMediaId)
        <x-picture :media="$author->avatarMediaId" :alt="$author->name" sizes="96px" class="size-24 rounded-full object-cover" picture-class="shrink-0"/>
    @endif
    <div class="flex flex-col gap-2.5">
        @if ($author->credentials)
            <p class="m-0 text-base font-bold text-ink"><span class="text-muted">{{ __('blog.author.credentials') }}:</span> {{ $author->credentials }}</p>
        @endif
        @if ($author->bio)
            <p class="m-0 text-lg leading-loose font-medium text-muted">{{ $author->bio }}</p>
        @endif
        @if ($author->sameAs !== [])
            <ul class="m-0 flex list-none flex-wrap gap-3 p-0">
                @foreach ($author->sameAs as $profile)
                    <li><a href="{{ $profile }}" rel="me noopener" class="text-sm-plus font-bold text-primary hover:text-primary-hover">{{ parse_url($profile, PHP_URL_HOST) ?: $profile }}</a></li>
                @endforeach
            </ul>
        @endif
    </div>
</div>
