/**
 * data-module="stepper" on the /directory/join form (L5-05). Without JS the form is one long page with every step
 * visible and a single «ارسال درخواست» button; this module turns it into steps:
 *
 *  - shows one [data-step] at a time (start: data-start — the first step with a server error — or #join-step-N),
 *    swaps the matching pre-rendered [data-stepper-nav] progress list, and toggles the footer buttons
 *    ([data-stepper-back] link on step 1, [data-stepper-prev], [data-stepper-next], [data-stepper-submit] on the last);
 *  - «ذخیره و ادامه» (and Enter) only moves on when the current step's controls pass the browser's constraint checks;
 *  - checks selected photos ([data-photos]: data-max count, data-bytes per file, data-total for all, data-types)
 *    before upload, reporting in [data-photos-status] and blocking the step via setCustomValidity;
 *  - limits the district list to the chosen city ([data-city-select] / optgroup[data-city]);
 *  - mirrors the typed name ([data-preview-name]) into the preview card ([data-preview]).
 * The server validates everything again (App\Http\Requests\JoinRequest).
 */
const fill = (template, values) => Object.entries(values).reduce((text, [key, value]) => text.replaceAll(`:${key}`, value), template);

const DIGITS = '۰۱۲۳۴۵۶۷۸۹';
const faDigits = (value) => String(value).replace(/\d/g, (d) => DIGITS[Number(d)]);

const controlsOf = (root) => [...root.querySelectorAll('input, select, textarea')].filter((el) => !el.disabled && el.type !== 'hidden');

function photoChecks(input, status) {
    const { max, bytes, total, types } = input.dataset;
    const allowed = (types || '').split(',').filter(Boolean);
    const check = () => {
        const files = [...input.files];
        let error = '';
        if (files.length > Number(max)) {
            error = input.dataset.msgTooMany;
        } else {
            const wrongType = files.find((file) => allowed.length && !allowed.includes(file.type));
            const tooBig = files.find((file) => file.size > Number(bytes));
            const sum = files.reduce((acc, file) => acc + file.size, 0);
            if (wrongType) error = fill(input.dataset.msgType, { name: wrongType.name });
            else if (tooBig) error = fill(input.dataset.msgTooBig, { name: tooBig.name });
            else if (Number(total) > 0 && sum > Number(total)) error = input.dataset.msgTotal;
        }
        input.setCustomValidity(error);
        if (status) {
            status.textContent = error || (files.length ? fill(input.dataset.msgSelected, { count: faDigits(files.length) }) : '');
            status.classList.toggle('text-danger', error !== '');
            status.classList.toggle('text-muted', error === '');
        }
    };
    input.addEventListener('change', check);
    check();
}

function districtFilter(city, district) {
    const sync = () => {
        district.querySelectorAll('optgroup[data-city]').forEach((group) => {
            const match = !city.value || group.dataset.city === city.value;
            group.hidden = !match;
            group.disabled = !match;
        });
        if (district.selectedOptions[0]?.parentElement?.disabled) district.value = '';
    };
    city.addEventListener('change', sync);
    sync();
}

export default (form) => {
    const steps = [...form.querySelectorAll('[data-step]')];
    if (steps.length < 2) return;

    const navs = [...form.querySelectorAll('[data-stepper-nav]')];
    const back = form.querySelector('[data-stepper-back]');
    const prev = form.querySelector('[data-stepper-prev]');
    const next = form.querySelector('[data-stepper-next]');
    const submit = form.querySelector('[data-stepper-submit]');
    const total = steps.length;

    const fromHash = /^#join-step-(\d)$/.exec(window.location.hash);
    let current = Number(form.dataset.start) || 1;
    if (current === 1 && fromHash) current = Math.min(total, Math.max(1, Number(fromHash[1])));

    // In the long (no-JS) form steps 2… are separated by a rule; a single visible step needs none.
    steps.forEach((step) => step.classList.remove('border-t', 'pt-5.5'));

    const show = (n, focus) => {
        current = n;
        steps.forEach((step) => {
            step.hidden = Number(step.dataset.step) !== n;
        });
        navs.forEach((nav) => {
            nav.hidden = Number(nav.dataset.stepperNav) !== n;
        });
        if (back) back.hidden = n !== 1;
        if (prev) prev.hidden = n === 1;
        if (next) next.hidden = n === total;
        if (submit) submit.hidden = n !== total;
        if (focus) {
            const heading = steps[n - 1].querySelector('h2');
            if (heading) {
                heading.tabIndex = -1;
                heading.focus({ preventScroll: true });
            }
            form.scrollIntoView({ block: 'start', behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' });
        }
    };

    const stepIsValid = () => {
        const invalid = controlsOf(steps[current - 1]).find((el) => !el.checkValidity());
        if (!invalid) return true;
        invalid.reportValidity();
        invalid.focus();
        return false;
    };

    next?.addEventListener('click', () => {
        if (stepIsValid()) show(Math.min(total, current + 1), true);
    });
    prev?.addEventListener('click', () => show(Math.max(1, current - 1), true));

    form.addEventListener('submit', (event) => {
        if (current < total) {
            event.preventDefault();
            if (stepIsValid()) show(current + 1, true);
            return;
        }
        const broken = steps.findIndex((step) => controlsOf(step).some((el) => !el.checkValidity()));
        if (broken !== -1) {
            event.preventDefault();
            show(broken + 1, false);
            stepIsValid();
            return;
        }
        if (submit) submit.disabled = true;
    });

    const photos = form.querySelector('[data-photos]');
    if (photos) photoChecks(photos, form.querySelector('[data-photos-status]'));

    const city = form.querySelector('[data-city-select]');
    const district = form.querySelector('[data-district-select]');
    if (city && district) districtFilter(city, district);

    const nameInput = form.querySelector('[data-preview-name]');
    const previewName = form.querySelector('[data-preview] a');
    if (nameInput && previewName) {
        const fallback = previewName.textContent;
        nameInput.addEventListener('input', () => {
            previewName.textContent = nameInput.value.trim() || fallback;
        });
    }

    show(current, false);
};
