<?php

declare(strict_types=1);

namespace App\Http\Controllers\Blog;

use App\Domain\Newsletter\Actions\ConfirmSubscription;
use App\Domain\Newsletter\Actions\Subscribe;
use App\Domain\Newsletter\Actions\Unsubscribe;
use App\Domain\Newsletter\Enums\SubscriptionStatus;
use App\Domain\Seo\SeoManager;
use Illuminate\Contracts\Validation\Factory as ValidatorFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

/**
 * Newsletter sign-up (double opt-in) from the magazine lists. Anti-spam without captcha: the `website` honeypot
 * (filled → silently dropped, same answer) and `throttle:newsletter` on the route. Every outcome shows the same
 * neutral message, so the form never reveals whether an address is subscribed. Result pages are noindex and never
 * page-cached (routes/web.php).
 */
final class NewsletterController
{
    public const HONEYPOT = 'website';

    public function store(Request $request, Subscribe $subscribe, ValidatorFactory $validator): RedirectResponse
    {
        $back = $this->backUrl($request);

        if (trim((string) $request->input(self::HONEYPOT, '')) !== '') {
            return redirect()->to($back)->with('newsletter_status', __('blog.newsletter.sent'));
        }

        // email:rfc only — `dns` would be an external request.
        $validator = $validator->make($request->only('email'), [
            'email' => ['required', 'string', 'max:191', 'email:rfc'],
        ], [
            'email.required' => __('blog.newsletter.errors.required'),
            'email.string' => __('blog.newsletter.errors.email'),
            'email.email' => __('blog.newsletter.errors.email'),
            'email.max' => __('blog.newsletter.errors.max'),
        ]);
        if ($validator->fails()) {
            return redirect()->to($back)->withErrors($validator, 'newsletter')->withInput($request->only('email'));
        }

        $subscribe->handle((string) $validator->validated()['email'], $this->source($request));

        return redirect()->to($back)->with('newsletter_status', __('blog.newsletter.sent'));
    }

    public function confirm(string $token, ConfirmSubscription $confirm, SeoManager $seo): View
    {
        $status = $confirm->handle($token);
        $ok = $status === SubscriptionStatus::Active;
        $title = __($ok ? 'blog.newsletter.confirmed.title' : 'blog.newsletter.invalid.title');

        $seo->title($title)->noindex();

        return view('pages.blog.newsletter.result', [
            'title' => $title,
            'text' => __($ok ? 'blog.newsletter.confirmed.text' : 'blog.newsletter.invalid.text'),
            'tone' => $ok ? 'success' : 'lavender',
            'icon' => $ok ? 'check' : 'info',
            'navRoute' => 'blog.index',
        ]);
    }

    public function unsubscribeForm(string $token, SeoManager $seo): View
    {
        $seo->title(__('blog.newsletter.unsubscribe.title'))->noindex();

        return view('pages.blog.newsletter.unsubscribe', [
            'action' => route('newsletter.unsubscribe.store', [$token]),
            'navRoute' => 'blog.index',
        ]);
    }

    /**
     * Form button and RFC 8058 one-click (List-Unsubscribe-Post) — the token is the authorisation, so no CSRF.
     */
    public function unsubscribe(string $token, Unsubscribe $unsubscribe, SeoManager $seo): View
    {
        $ok = $unsubscribe->handle($token);
        $title = __($ok ? 'blog.newsletter.unsubscribe.done_title' : 'blog.newsletter.invalid.title');

        $seo->title($title)->noindex();

        return view('pages.blog.newsletter.result', [
            'title' => $title,
            'text' => __($ok ? 'blog.newsletter.unsubscribe.done_text' : 'blog.newsletter.invalid.text'),
            'tone' => 'lavender',
            'icon' => $ok ? 'check' : 'info',
            'navRoute' => 'blog.index',
        ]);
    }

    /**
     * Magazine path the form was posted from (`blog`, `blog/category/cycle` …), validated so it can never redirect
     * off-site.
     */
    private function source(Request $request): ?string
    {
        $source = trim((string) $request->input('source', ''), '/');

        return $source !== '' && mb_strlen($source) <= 100 && preg_match('#^blog(/[\pL\pN_\-]+){0,2}$#u', $source) === 1 ? $source : null;
    }

    private function backUrl(Request $request): string
    {
        $source = $this->source($request);

        return ($source === null ? route('blog.index') : url($source)).'#newsletter';
    }
}
