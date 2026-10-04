/**
 * data-module="faq-filter" on the /faq search input (aria-controls = the categories container). Hides questions that
 * don't contain every typed word (question + answer, Arabic/Persian letters, digits and ZWNJ normalised), opens the
 * matching <details>, hides empty categories and shows the [data-faq-empty] note when nothing matches. Clearing the
 * box restores the page. Also keeps aria-current on the side-nav link of the category the visitor jumps to.
 * Without JS every question is visible (the input does nothing).
 */
const DIGITS = { '۰': '0', '۱': '1', '۲': '2', '۳': '3', '۴': '4', '۵': '5', '۶': '6', '۷': '7', '۸': '8', '۹': '9' };

const normalize = (text) =>
    text
        .toLowerCase()
        .replace(/[يى]/g, 'ی')
        .replace(/ك/g, 'ک')
        .replace(/[۰-۹]/g, (d) => DIGITS[d])
        .replace(/[ً-ٟـ]/g, '')
        .replace(/[‌\s]+/g, ' ')
        .trim();

export default (input) => {
    const root = document.getElementById(input.getAttribute('aria-controls'));
    if (!root) return;

    const groups = [...root.querySelectorAll('[data-faq-group]')];
    const items = groups.flatMap((group) =>
        [...group.querySelectorAll('[data-faq-item]')].map((element) => ({
            element,
            group,
            text: normalize(element.textContent),
            open: element.open === true,
        })),
    );
    const empty = root.querySelector('[data-faq-empty]');

    const filter = () => {
        const words = normalize(input.value).split(' ').filter(Boolean);
        const visible = new Set();
        items.forEach((item) => {
            const match = words.every((word) => item.text.includes(word));
            item.element.hidden = !match;
            if ('open' in item.element) item.element.open = words.length ? match : item.open;
            if (match) visible.add(item.group);
        });
        groups.forEach((group) => {
            group.hidden = !visible.has(group);
        });
        if (empty) empty.hidden = visible.size > 0;
    };
    input.addEventListener('input', filter);
    input.addEventListener('search', filter);

    const links = [...document.querySelectorAll('a[href^="#"]')].filter((link) =>
        groups.some((group) => link.getAttribute('href') === `#${group.id}`),
    );
    const mark = (hash) => {
        if (!links.some((link) => link.getAttribute('href') === hash)) return;
        links.forEach((link) => {
            if (link.getAttribute('href') === hash) link.setAttribute('aria-current', 'true');
            else link.removeAttribute('aria-current');
        });
    };
    window.addEventListener('hashchange', () => mark(location.hash));
    mark(location.hash);
};
