/**
 * data-module="cart-badge" on the shop cart button (components/shop/cart-link, L6-02). Shop pages are full-page
 * cached for guests, so the item count is never rendered by the server: this module reads the `ritme_cart_count`
 * cookie (a plain integer the cart of L6-04 writes, not HttpOnly, excluded from cookie encryption) and fills the
 * `[data-cart-count]` badge with Persian digits, updating the button's aria-label from data-label-count
 * («:count کالا در سبد»). No cookie, 0 or junk → the badge stays hidden.
 */
const COOKIE = 'ritme_cart_count';
const MAX = 99;

const readCount = () => {
    const pair = document.cookie.split('; ').find((part) => part.startsWith(`${COOKIE}=`));
    const value = pair ? Number.parseInt(decodeURIComponent(pair.slice(COOKIE.length + 1)), 10) : 0;

    return Number.isFinite(value) && value > 0 ? value : 0;
};

const persian = (value) => String(value).replace(/\d/g, (digit) => '۰۱۲۳۴۵۶۷۸۹'[Number(digit)]);

export default function cartBadge(element) {
    const badge = element.querySelector('[data-cart-count]');
    const count = readCount();

    if (!badge || count === 0) {
        return;
    }

    const text = count > MAX ? `${persian(MAX)}+` : persian(count);
    badge.textContent = text;
    badge.hidden = false;

    const template = element.dataset.labelCount;
    if (template) {
        element.setAttribute('aria-label', `${element.dataset.label ?? ''} — ${template.replace(':count', text)}`);
    }
}
