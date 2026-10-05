/**
 * data-module="cart" — lazy enhancement of the cart forms (L6-04). Everything works without it (POST + redirect).
 *
 * Cart page (`pages/shop/cart`, root = the page section): forms marked `data-cart-form` (− / + / حذف) are sent with
 * fetch; the redirect is followed and the fresh page's `[data-cart-body]`, status and error texts replace the current
 * ones (one source of markup: the server). Focus returns to the same control when it still exists, else to the
 * status line.
 *
 * Product page (root = the add-to-cart `<form data-cart-slot>`): the form is posted with `Accept: application/json`;
 * the answer {ok, message, count} is shown in the form's `[data-cart-toast]` (role=status) with its «دیدن سبد» link.
 *
 * Both update the header badge from the `ritme_cart_count` cookie the server just wrote. Any network / unexpected
 * answer falls back to a normal form submission.
 */
import cartBadge from './cart-badge.js';

const refreshBadges = () => {
    document.querySelectorAll('[data-module~="cart-badge"]').forEach((element) => {
        const badge = element.querySelector('[data-cart-count]');
        if (badge) {
            badge.hidden = true;
            badge.textContent = '';
        }
        if (element.dataset.label) element.setAttribute('aria-label', element.dataset.label);
        cartBadge(element);
    });
};

const nativeSubmit = (form, submitter) => {
    if (submitter?.name) {
        const input = document.createElement('input');
        input.type = 'hidden';
        input.name = submitter.name;
        input.value = submitter.value;
        form.append(input);
    }
    HTMLFormElement.prototype.submit.call(form);
};

const setMessage = (node, text) => {
    if (!node) return;
    node.textContent = text ?? '';
    node.classList.toggle('hidden', !node.textContent.trim());
};

const enhanceProduct = (form) => {
    const toast = form.querySelector('[data-cart-toast]');
    const toastText = toast?.querySelector('[data-cart-toast-text]');

    form.addEventListener('submit', async (event) => {
        if (!form.getAttribute('action')) return;
        event.preventDefault();
        const submit = form.querySelector('[data-cart-submit]');
        if (submit) submit.setAttribute('aria-busy', 'true');

        try {
            const response = await fetch(form.action, {
                method: 'POST',
                body: new FormData(form),
                headers: { Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
                credentials: 'same-origin',
            });
            const data = await response.json();
            if (typeof data?.message !== 'string') throw new Error('unexpected answer');

            if (toast && toastText) {
                toastText.textContent = data.message;
                toast.dataset.state = data.ok ? 'ok' : 'error';
                toast.querySelector('[data-cart-toast-link]')?.toggleAttribute('hidden', !data.ok);
                toast.classList.remove('hidden');
            }
            refreshBadges();
        } catch {
            nativeSubmit(form, event.submitter);
        } finally {
            if (submit) submit.removeAttribute('aria-busy');
        }
    });
};

const enhanceCartPage = (root) => {
    root.addEventListener('submit', async (event) => {
        const form = event.target;
        if (!(form instanceof HTMLFormElement) || !form.matches('[data-cart-form]')) return;
        event.preventDefault();

        const submitter = event.submitter;
        const focusKey = submitter ? [form.getAttribute('action'), submitter.getAttribute('aria-label')] : null;
        const body = new FormData(form);
        if (submitter?.name) body.set(submitter.name, submitter.value);
        root.setAttribute('aria-busy', 'true');

        try {
            const response = await fetch(form.action, {
                method: 'POST',
                body,
                headers: { Accept: 'text/html', 'X-Requested-With': 'XMLHttpRequest' },
                credentials: 'same-origin',
            });
            if (!response.ok || !response.redirected) throw new Error(`HTTP ${response.status}`);

            const doc = new DOMParser().parseFromString(await response.text(), 'text/html');
            const fresh = doc.querySelector('[data-cart-body]');
            const current = root.querySelector('[data-cart-body]');
            if (!fresh || !current) throw new Error('no cart body');

            current.replaceChildren(...fresh.childNodes);
            setMessage(root.querySelector('[data-cart-status]'), doc.querySelector('[data-cart-status]')?.textContent);
            setMessage(root.querySelector('[data-cart-error]'), doc.querySelector('[data-cart-error]')?.textContent);
            refreshBadges();

            const again = focusKey
                ? [...root.querySelectorAll('[data-cart-form]')]
                      .find((f) => f.getAttribute('action') === focusKey[0])
                      ?.querySelector(`button[aria-label="${CSS.escape(focusKey[1] ?? '')}"]:not([disabled])`)
                : null;
            if (again) {
                again.focus();
            } else {
                const status = root.querySelector('[data-cart-status]:not(.hidden)') ?? root.querySelector('h1');
                status?.setAttribute('tabindex', '-1');
                status?.focus();
            }
        } catch {
            nativeSubmit(form, submitter);
        } finally {
            root.removeAttribute('aria-busy');
        }
    });
};

export default function cart(element) {
    if (element instanceof HTMLFormElement) {
        enhanceProduct(element);
    } else {
        enhanceCartPage(element);
    }
}
