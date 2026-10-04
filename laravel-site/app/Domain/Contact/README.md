# Contact

Contact form submissions (admin inbox) and their notification flow (L3-10).

- `Models/ContactMessage` (table `contact_messages`; minimal data: topic, name, email **or** phone, message, status).
- `Actions/SubmitContactMessage` (store + queued `App\Notifications\ContactMessageReceived` to the topic's mailbox,
  `Support/ContactRecipients`), `ChangeContactMessageStatus` (unread/read/archived, activity log `contact`),
  `ExportContactMessages` (CSV).
- Anti-spam without captcha: honeypot + `Support/FormTimer` (time trap) + named rate limiter `contact`
  (`ContactServiceProvider`). `Support/ReplyChannel` parses the «ایمیل یا شماره همراه» field.

Bindings live in `App\Providers\Domain\ContactServiceProvider`. See `docs/ARCHITECTURE.md`.
