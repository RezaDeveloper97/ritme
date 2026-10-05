<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Orders;

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Shop\Ordering\Models\Order;
use App\Filament\Resources\Shop\ShopAdmin;
use Filament\Facades\Filament;
use Illuminate\Http\Response;
use Illuminate\Support\Facades\Gate;

/**
 * GET {admin}/shop/orders/{record}/invoice — the printable invoice / packing slip of an order (Persian, RTL, Jalali,
 * tomans). A standalone page with its own small stylesheet: no Vite build, no fonts, no external request. Registered
 * by OrderResource::registerRoutes() inside the panel's auth middleware. Access: OrderPolicy::view().
 */
final class OrderInvoiceController
{
    public function __invoke(string $record, SettingsRepository $settings): Response
    {
        $order = Order::query()->with('items')->findOrFail((int) $record);
        Gate::forUser(Filament::auth()->user())->authorize('view', $order);

        $all = $settings->all();

        $items = [];
        foreach ($order->items as $item) {
            $items[] = [
                'title' => $item->title,
                'variant' => $item->variant_label,
                'quantity' => fa_digits($item->quantity),
                'unit' => $item->unit_price->format(),
                'total' => $item->line_total->format(),
            ];
        }

        $invoice = [
            'seller' => $all->general->siteName !== '' ? $all->general->siteName : 'ریتمی',
            'sellerPhone' => $all->contact->phone,
            'code' => $order->code,
            'placedAt' => ShopAdmin::date($order->created_at) ?? '',
            'status' => $order->status->label(),
            'payment' => $order->payment_method->label().' — '.$order->payment_status->label(),
            'recipient' => $order->recipient_name,
            'mobile' => fa_digits($order->mobile),
            'region' => $order->province.' / '.$order->city,
            'address' => $order->address,
            'postalCode' => $order->postal_code === null ? null : fa_digits($order->postal_code),
            'delivery' => trim((ShopAdmin::date($order->delivery_date, 'l j F Y') ?? '').' — ساعت '.$order->delivery_window->hours()),
            'discreet' => $order->discreet_packaging,
            'note' => $order->note,
            'items' => $items,
            'subtotal' => $order->subtotal->format(),
            'shipping' => $order->shipping_fee?->format(),
            'total' => $order->total->format(),
            'isDemo' => $order->is_demo,
        ];

        return response()
            ->view('admin.invoice', ['invoice' => $invoice])
            ->header('Cache-Control', 'private, no-store')
            ->header('X-Robots-Tag', 'noindex, nofollow');
    }
}
