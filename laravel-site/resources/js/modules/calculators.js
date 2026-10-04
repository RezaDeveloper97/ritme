/**
 * data-module="calculators" on each /tools calculator <form> (data-kind="due"|"fert").
 *
 * Without JS the form is a plain GET to /tools (server renders the result). With JS the submit is answered in place
 * with the same maths and words (resources/js/lib/jalali.js + the form's data-text / data-errors, which hold the
 * lang/fa/tools.php `text` / `errors` arrays): the error goes into [data-calc-error] with aria-invalid on the field,
 * the result into [data-calc-result] (aria-live). Nothing is stored or sent anywhere.
 */
import { evaluate, today } from '../lib/jalali.js';

const json = (value) => {
    try {
        return JSON.parse(value || '{}');
    } catch {
        return {};
    }
};

export default (form) => {
    const templates = json(form.dataset.text);
    const messages = json(form.dataset.errors);
    const kind = form.dataset.kind;
    const fields = { lmp: form.elements.namedItem('lmp'), cycle: form.elements.namedItem('cycle') };
    const error = form.querySelector('[data-calc-error]');
    const result = form.querySelector('[data-calc-result]');
    const value = form.querySelector('[data-calc-value]');
    const detail = form.querySelector('[data-calc-detail]');
    if (!fields.lmp || !fields.cycle || !error || !result || !value || !detail) return;

    const mark = (field, invalid) => {
        if (invalid) field.setAttribute('aria-invalid', 'true');
        else field.removeAttribute('aria-invalid');
        const box = field.parentElement;
        box.classList.toggle('border-danger', invalid);
        box.classList.toggle('border-line', !invalid);
    };

    form.addEventListener('submit', (event) => {
        event.preventDefault();
        const outcome = evaluate(kind, fields.lmp.value, fields.cycle.value, templates, today());
        const field = outcome.error === 'cycle_range' ? 'cycle' : 'lmp';

        mark(fields.lmp, Boolean(outcome.error) && field === 'lmp');
        mark(fields.cycle, Boolean(outcome.error) && field === 'cycle');

        if (outcome.error) {
            error.textContent = messages[outcome.error] || '';
            result.hidden = true;
            fields[field].focus();
            return;
        }

        error.textContent = '';
        value.textContent = outcome.value;
        detail.textContent = outcome.detail;
        result.hidden = false;
    });
};
