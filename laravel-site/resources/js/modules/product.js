/**
 * data-module="product" on the product page's buy box (pages/shop/product, L6-03). The server renders a working
 * no-JS form: colour / size radios (a size is disabled only when sold out in every colour) and a number input. This
 * module refines it for the selected colour:
 *   - sizes without stock in that colour are struck through and disabled; a now-disabled selection moves to the
 *     first available size,
 *   - «رنگ: …», the price (+ compare-at), the stock badge, the size hint, the highlighted size-chart row and the
 *     quantity limit follow the selected variant,
 *   - the − / + buttons (hidden without JS) step the quantity within 1…max.
 * Variant data comes from `data-variants` ([{id, size, color, price, compare, stock, max}], prices preformatted),
 * size hints from `data-hints`; all copy lives in the markup. The cart submit (fetch + toast) is L6-04's.
 */
const parse = (value, fallback) => {
    try {
        return JSON.parse(value || '');
    } catch {
        return fallback;
    }
};

const toLatin = (s) => String(s).replace(/[۰-۹]/g, (d) => '۰۱۲۳۴۵۶۷۸۹'.indexOf(d)).replace(/[٠-٩]/g, (d) => '٠١٢٣٤٥٦٧٨٩'.indexOf(d));

const TONES = { in: ['bg-success-soft', 'text-success'], out: ['bg-danger-soft', 'text-danger'] };

export default (root) => {
    const variants = parse(root.dataset.variants, []);
    const hints = parse(root.dataset.hints, {});
    const colors = [...root.querySelectorAll('input[data-color]')];
    const sizes = [...root.querySelectorAll('input[data-size]')];
    const qty = root.querySelector('input[data-qty]');
    const submit = root.querySelector('[data-cart-submit]');
    const cartLabel = root.querySelector('[data-cart-label]');
    const canSubmit = Boolean(root.querySelector('form[action]'));

    const checked = (inputs) => inputs.find((input) => input.checked)?.value ?? null;
    const find = (size, color) =>
        variants.find((v) => (sizes.length === 0 || v.size === size) && (colors.length === 0 || v.color === color)) ?? null;
    const hasStock = (size, color) => variants.some((v) => v.size === size && (colors.length === 0 || v.color === color) && v.stock);

    const setText = (selector, text) => {
        const node = root.querySelector(selector);
        if (node) node.textContent = text;
    };

    const clampQty = (max) => {
        if (!qty) return;
        qty.max = String(max);
        const value = Number.parseInt(toLatin(qty.value), 10);
        qty.value = String(Math.min(Math.max(Number.isFinite(value) ? value : 1, 1), max));
    };

    const update = () => {
        const color = checked(colors);

        sizes.forEach((input) => {
            const available = hasStock(input.value, color);
            input.disabled = !available;
            input.closest('label')?.toggleAttribute('data-soldout', !available);
        });
        if (sizes.length > 0 && !sizes.some((input) => input.checked && !input.disabled)) {
            const first = sizes.find((input) => !input.disabled);
            if (first) first.checked = true;
        }

        const size = checked(sizes);
        const variant = find(size, color);
        const inStock = Boolean(variant?.stock);

        setText('[data-color-name]', color ?? '');
        if (variant) {
            setText('[data-price]', variant.price);
            setText('[data-compare-amount]', variant.compare ?? '');
            root.querySelector('[data-compare]')?.classList.toggle('hidden', !variant.compare);
        }

        const badge = root.querySelector('[data-stock]');
        if (badge) {
            badge.textContent = inStock ? root.dataset.labelIn : root.dataset.labelOut;
            badge.classList.remove(...TONES.in, ...TONES.out);
            badge.classList.add(...(inStock ? TONES.in : TONES.out));
        }

        const hint = root.querySelector('[data-size-hint]');
        if (hint) {
            hint.textContent = (size && hints[size]) || '';
            hint.classList.toggle('hidden', !hint.textContent);
        }

        document.querySelectorAll('[data-size-row]').forEach((row) => {
            row.toggleAttribute('data-current', row.dataset.sizeRow === size);
        });

        clampQty(variant?.max ?? Number.parseInt(qty?.max ?? '1', 10));

        if (cartLabel) cartLabel.textContent = inStock ? root.dataset.labelAdd : root.dataset.labelUnavailable;
        if (submit && canSubmit) submit.disabled = !inStock;
    };

    root.querySelectorAll('[data-qty-step]').forEach((button) => {
        button.hidden = false;
        button.addEventListener('click', () => {
            if (!qty) return;
            const value = Number.parseInt(toLatin(qty.value), 10) || 1;
            const max = Number.parseInt(qty.max, 10) || 1;
            qty.value = String(Math.min(Math.max(value + Number(button.dataset.qtyStep), 1), max));
            qty.dispatchEvent(new Event('change', { bubbles: true }));
        });
    });
    qty?.addEventListener('change', () => clampQty(Number.parseInt(qty.max, 10) || 1));

    [...colors, ...sizes].forEach((input) => input.addEventListener('change', update));

    if (variants.length > 0) update();
};
