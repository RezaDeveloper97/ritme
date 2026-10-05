{{--
    Printable invoice / packing slip of a shop order (admin only, L6-06). Standalone: its own tiny stylesheet, system
    fonts, no Vite build, no external request. Data arrives as plain values from OrderInvoiceController.
--}}
<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="robots" content="noindex, nofollow">
    <title>فاکتور سفارش {{ $invoice['code'] }}</title>
    <style>
        * { box-sizing: border-box; }
        body { margin: 0; padding: 24px; font-family: Vazirmatn, Tahoma, "Segoe UI", sans-serif; color: black; background: white; font-size: 14px; line-height: 1.7; }
        .sheet { max-width: 800px; margin: 0 auto; }
        header { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; border-bottom: 2px solid black; padding-bottom: 12px; margin-bottom: 16px; }
        h1 { font-size: 20px; margin: 0 0 4px; }
        .muted { color: dimgray; }
        .full { grid-column: 1 / -1; }
        .ltr { direction: ltr; unicode-bidi: embed; }
        .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4px 24px; margin-bottom: 16px; }
        .grid div span:first-child { color: dimgray; margin-inline-end: 6px; }
        table { width: 100%; border-collapse: collapse; margin-bottom: 16px; }
        th, td { border: 1px solid silver; padding: 6px 8px; text-align: start; vertical-align: top; }
        th { background: whitesmoke; font-weight: 600; }
        td.num, th.num { text-align: end; white-space: nowrap; }
        .totals { width: 50%; margin-inline-start: auto; }
        .totals td { border: none; border-bottom: 1px solid gainsboro; }
        .totals tr:last-child td { font-weight: 700; font-size: 16px; border-bottom: 2px solid black; }
        .box { border: 1px dashed gray; padding: 8px 12px; margin-bottom: 16px; }
        .actions { text-align: center; margin-top: 24px; }
        .actions button { font: inherit; padding: 8px 24px; cursor: pointer; }
        .demo { border: 2px solid black; padding: 4px 12px; display: inline-block; font-weight: 700; }
        @media print {
            body { padding: 0; }
            .actions { display: none; }
        }
    </style>
</head>
<body>
<main class="sheet">
    <header>
        <div>
            <h1>{{ $invoice['seller'] }} — فاکتور فروش</h1>
            @if ($invoice['sellerPhone'])
                <div class="muted">تلفن پشتیبانی: <span class="ltr">{{ fa_digits($invoice['sellerPhone']) }}</span></div>
            @endif
            @if ($invoice['isDemo'])
                <div class="demo">سفارش نمونه (نمایشی)</div>
            @endif
        </div>
        <div>
            <div>کد سفارش: <strong class="ltr">{{ $invoice['code'] }}</strong></div>
            <div class="muted">ثبت: {{ $invoice['placedAt'] }}</div>
            <div class="muted">وضعیت: {{ $invoice['status'] }}</div>
        </div>
    </header>

    <section class="grid" aria-label="گیرنده">
        <div><span>گیرنده:</span><span>{{ $invoice['recipient'] }}</span></div>
        <div><span>موبایل:</span><span class="ltr">{{ $invoice['mobile'] }}</span></div>
        <div><span>استان / شهر:</span><span>{{ $invoice['region'] }}</span></div>
        <div><span>کد پستی:</span><span class="ltr">{{ $invoice['postalCode'] ?? '—' }}</span></div>
        <div class="full"><span>نشانی:</span><span>{{ $invoice['address'] }}</span></div>
        <div><span>تحویل ترجیحی:</span><span>{{ $invoice['delivery'] }}</span></div>
        <div><span>پرداخت:</span><span>{{ $invoice['payment'] }}</span></div>
    </section>

    @if ($invoice['discreet'])
        <div class="box">بسته‌بندی بدون نام و تصویر محصول روی جعبه.</div>
    @endif

    <table>
        <thead>
        <tr>
            <th>#</th>
            <th>کالا</th>
            <th class="num">تعداد</th>
            <th class="num">قیمت واحد</th>
            <th class="num">جمع</th>
        </tr>
        </thead>
        <tbody>
        @foreach ($invoice['items'] as $i => $item)
            <tr>
                <td>{{ fa_digits($i + 1) }}</td>
                <td>
                    {{ $item['title'] }}
                    @if ($item['variant'])
                        <div class="muted">{{ $item['variant'] }}</div>
                    @endif
                </td>
                <td class="num">{{ $item['quantity'] }}</td>
                <td class="num">{{ $item['unit'] }}</td>
                <td class="num">{{ $item['total'] }}</td>
            </tr>
        @endforeach
        </tbody>
    </table>

    <table class="totals">
        <tr><td>جمع کالاها</td><td class="num">{{ $invoice['subtotal'] }}</td></tr>
        <tr><td>هزینه ارسال</td><td class="num">{{ $invoice['shipping'] ?? 'هنگام تماس اعلام می‌شود' }}</td></tr>
        <tr><td>مبلغ قابل پرداخت هنگام تحویل</td><td class="num">{{ $invoice['total'] }}</td></tr>
    </table>

    @if ($invoice['note'])
        <div class="box"><strong>توضیح مشتری:</strong> {{ $invoice['note'] }}</div>
    @endif

    <div class="actions"><button type="button" onclick="window.print()">چاپ</button></div>
</main>
</body>
</html>
