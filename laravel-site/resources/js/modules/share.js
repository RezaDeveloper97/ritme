/**
 * data-module="share" on the article share block (components/blog/share). Reveals the hidden [data-share-copy]
 * button and copies the [data-share-url] field to the clipboard, announcing the result in [data-share-status]
 * (Persian copy from data-share-copied / data-share-failed). Falls back to selecting the field so the reader can
 * copy by hand. Without JS the read-only field alone does the job.
 */
const copyText = async (input) => {
    if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(input.value);
        return;
    }
    input.select();
    if (!document.execCommand('copy')) throw new Error('copy failed');
};

export default (root) => {
    const input = root.querySelector('[data-share-url]');
    const button = root.querySelector('[data-share-copy]');
    const status = root.querySelector('[data-share-status]');
    if (!input || !button) return;

    let timer;
    const say = (text) => {
        if (!status) return;
        status.textContent = text;
        clearTimeout(timer);
        timer = setTimeout(() => {
            status.textContent = '';
        }, 4000);
    };

    input.addEventListener('focus', () => input.select());
    button.hidden = false;
    button.addEventListener('click', async () => {
        try {
            await copyText(input);
            say(root.dataset.shareCopied || '');
        } catch {
            input.focus();
            input.select();
            say(root.dataset.shareFailed || '');
        }
    });
};
