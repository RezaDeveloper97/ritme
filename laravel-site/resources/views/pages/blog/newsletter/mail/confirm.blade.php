{{-- Double opt-in mail (HTML part). Plain markup on purpose: mail clients, no site CSS, no external images. --}}
<!doctype html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ __('blog.newsletter.mail.subject') }}</title>
</head>
<body dir="rtl">
    <p>{{ __('blog.newsletter.mail.greeting') }}</p>
    <p>{{ __('blog.newsletter.mail.body') }}</p>
    <p><a href="{{ $confirmUrl }}"><strong>{{ __('blog.newsletter.mail.button') }}</strong></a></p>
    <p>{{ __('blog.newsletter.mail.ignore') }}</p>
    <p>{{ __('blog.newsletter.mail.unsubscribe') }} <a href="{{ $unsubscribeUrl }}">{{ $unsubscribeUrl }}</a></p>
    <p>{{ __('blog.newsletter.mail.signature') }}</p>
</body>
</html>
