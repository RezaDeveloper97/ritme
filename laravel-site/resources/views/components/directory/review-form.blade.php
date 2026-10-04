{{--
    <x-directory.review-form :action="route('directory.place.review', $slug)" :aspects="[['key' => 'staff', 'label' => 'مربی']]" honeypot="company_url"/>
    Review form of the place page (L5-03), folded in a native <details> (opens by itself after a failed submit). Posts
    to ShowPlaceController::storeReview: moderated (pending until approved), rate limited, with an off-screen honeypot.
    `@csrf` keeps working on page-cache HITs (PageCache swaps the token per visitor). Errors and the thank-you note
    come back as flash data, which bypasses the page cache.
--}}
@props(['action', 'aspects' => [], 'honeypot' => 'company_url'])
@php($failed = $errors->hasAny(['author_name', 'rating', 'body', 'aspects']) || $errors->has('aspects.*'))
<div id="review-form" class="flex flex-col gap-3">
    @if (session('review_submitted'))
        <p role="status" class="m-0 flex items-center gap-3 rounded-3xl border border-stage-teen/40 bg-stage-teen/12 px-5 py-4 text-base font-bold text-ink">
            <x-icon name="check" class="size-5 shrink-0 text-stage-teen"/>{{ __('directory.place.review.sent') }}
        </p>
    @endif
    <details class="group rounded-4xl border border-line bg-surface" @if ($failed) open @endif>
        <summary class="flex h-13 cursor-pointer list-none items-center gap-2 px-5 text-base font-extrabold text-primary [&::-webkit-details-marker]:hidden">
            <x-icon name="message" class="size-5"/>{{ __('directory.place.review.open') }}
            <x-icon name="chevron-left" class="ms-auto size-4 -rotate-90 text-muted transition-transform group-open:rotate-90"/>
        </summary>
        <form method="post" action="{{ $action }}" class="relative flex flex-col gap-4 px-5 pb-5">
            @csrf
            <p class="m-0 text-sm leading-relaxed font-semibold text-muted">{{ __('directory.place.review.intro') }}</p>
            @if ($failed)
                <p role="alert" class="m-0 rounded-2xl bg-danger-soft px-4 py-3 text-sm font-bold text-danger">{{ __('directory.place.review.errors.title') }}</p>
            @endif

            <x-ui.form.field for="review-name" :label="__('directory.place.review.name')" :hint="__('directory.place.review.name_hint')" :error="$errors->first('author_name')" required>
                <x-ui.form.input id="review-name" name="author_name" :value="old('author_name')" maxlength="80" autocomplete="nickname" required :invalid="$errors->has('author_name')" described/>
            </x-ui.form.field>

            <fieldset class="m-0 flex min-w-0 flex-col gap-2 border-0 p-0" @if ($errors->has('rating')) aria-describedby="review-rating-error" @endif>
                <legend class="mb-2 p-0 text-base font-extrabold text-ink">{{ __('directory.place.review.rating') }} <span aria-hidden="true" class="text-danger">*</span></legend>
                <div class="flex flex-wrap gap-2">
                    @for ($n = 5; $n >= 1; $n--)
                        <label class="flex h-11 cursor-pointer items-center gap-1.5 rounded-full border-[1.5px] border-line bg-surface px-4 text-base font-extrabold has-checked:border-primary has-checked:bg-primary/11 has-focus-visible:outline-2 has-focus-visible:outline-primary">
                            <input type="radio" name="rating" value="{{ $n }}" class="sr-only" required @checked((string) old('rating') === (string) $n)>
                            <x-icon name="star" class="size-4 text-stage-ttc"/>{{ __('directory.place.review.stars', ['n' => fa_digits($n)]) }}
                        </label>
                    @endfor
                </div>
                @if ($errors->has('rating'))<span id="review-rating-error" class="text-sm font-bold text-danger">{{ $errors->first('rating') }}</span>@endif
            </fieldset>

            @if ($aspects !== [])
                <fieldset class="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
                    <legend class="mb-2 p-0 text-base font-extrabold text-ink">{{ __('directory.place.review.aspects') }}</legend>
                    <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                        @foreach ($aspects as $aspect)
                            <label class="flex flex-col gap-1.5 text-sm font-bold text-muted">
                                {{ $aspect['label'] }}
                                <x-ui.form.select name="aspects[{{ $aspect['key'] }}]" :options="[5 => fa_digits(5), 4 => fa_digits(4), 3 => fa_digits(3), 2 => fa_digits(2), 1 => fa_digits(1)]"
                                                  :selected="old('aspects.'.$aspect['key'])" :placeholder="__('directory.place.review.aspect_none')" :invalid="$errors->has('aspects.'.$aspect['key'])"/>
                            </label>
                        @endforeach
                    </div>
                    @if ($errors->has('aspects') || $errors->has('aspects.*'))<span class="text-sm font-bold text-danger">{{ __('directory.place.review.errors.aspects') }}</span>@endif
                </fieldset>
            @endif

            <x-ui.form.field for="review-body" :label="__('directory.place.review.body')" :error="$errors->first('body')" required>
                <x-ui.form.textarea id="review-body" name="body" rows="4" maxlength="2000" required :invalid="$errors->has('body')" described>{{ old('body') }}</x-ui.form.textarea>
            </x-ui.form.field>

            {{-- Honeypot: off-screen, out of the tab order; a filled value is answered like a real submit and dropped. --}}
            <div aria-hidden="true" class="absolute -start-[9999px] top-0 size-px overflow-hidden">
                <label for="review-{{ $honeypot }}">{{ __('directory.place.review.honeypot') }}</label>
                <input type="text" id="review-{{ $honeypot }}" name="{{ $honeypot }}" value="" tabindex="-1" autocomplete="off">
            </div>

            <x-ui.button type="submit" size="md" class="self-start">{{ __('directory.place.review.submit') }}</x-ui.button>
        </form>
    </details>
</div>
