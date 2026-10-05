# Newsletter

Newsletter subscriptions (double opt-in, unsubscribe) — L4-02. Form «هر هفته یک خواندنی کوتاه» on the magazine lists.

- Table `newsletter_subscribers`, model `Models\Subscriber` (email, source, token, consent_at, confirmation_sent_at,
  confirmed_at, unsubscribed_at). Status is derived (`Enums\SubscriptionStatus`: pending / active / unsubscribed);
  `Subscriber::query()->active()` = may be mailed. No name, IP or user agent is stored.
- `Actions\Subscribe` (pending + `Mail\ConfirmSubscriptionMail`, at most one mail per 10 min while pending, same
  neutral answer for every outcome), `Actions\ConfirmSubscription`, `Actions\Unsubscribe` (idempotent, RFC 8058
  one-click target). The token is rotated on every new sign-up after an unsubscribe.
- Anti-spam without captcha/external services: honeypot field `website` (filled → silently dropped) and the
  `throttle:newsletter` limiter (3/min, 20/day per IP, `NewsletterServiceProvider`).
- Routes (`routes/web.php`): `newsletter.store` (POST), `newsletter.confirm` (GET → button page) + `newsletter.confirm.store` (POST, CSRF; F19), `newsletter.unsubscribe`
  (GET form / POST, CSRF-free for one-click). None of them is page-cached. Controller: `App\Http\Controllers\Blog\NewsletterController`.
- Mail driver: `MAIL_MAILER=log` by default. Admin list + CSV export: L4-05b.
- The confirmation mail is queued after commit (F16): needs the cron-driven queue worker in production.
