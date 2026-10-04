/**
 * data-module="gallery" on the place gallery (components/directory/gallery). Without JS, «همه N عکس» is a native
 * <details> listing every photo as a link to its large file. With JS that summary, a click on a mosaic image or on a
 * listed photo opens the server-rendered <dialog data-gallery-dialog> as a lightbox instead: one large image at a
 * time, previous / next buttons, arrow keys (RTL: ← next, → previous), Esc or «بستن» closes and focus returns to
 * the opener. All copy and classes live in the Blade markup; images load only when shown.
 */
export default (root) => {
    const dialog = root.querySelector('[data-gallery-dialog]');
    const items = [...root.querySelectorAll('[data-gallery-item]')];
    if (!dialog || items.length === 0 || typeof dialog.showModal !== 'function') return;

    const stage = dialog.querySelector('[data-gallery-stage]');
    const counter = dialog.querySelector('[data-gallery-counter]');
    const template = counter?.dataset.template || '';
    const toPersian = (n) => String(n).replace(/\d/g, (d) => '۰۱۲۳۴۵۶۷۸۹'[d]);

    let index = 0;
    let opener = null;

    const show = (next) => {
        index = (next + items.length) % items.length;
        const link = items[index];
        const thumb = link.querySelector('img');
        const image = document.createElement('img');
        image.src = link.href;
        image.alt = thumb?.alt || '';
        image.decoding = 'async';
        if (thumb?.width && thumb?.height) {
            image.width = thumb.width;
            image.height = thumb.height;
        }
        image.className = stage.dataset.galleryImgClass || '';
        stage.replaceChildren(image);
        if (counter) {
            counter.textContent = template.replace(':n', toPersian(index + 1)).replace(':total', toPersian(items.length));
        }
    };

    const open = (at, from) => {
        opener = from || document.activeElement;
        show(at);
        dialog.showModal();
    };

    root.querySelector('[data-gallery-open]')?.addEventListener('click', (event) => {
        event.preventDefault();
        open(0, event.currentTarget);
    });

    items.forEach((link, at) => {
        link.addEventListener('click', (event) => {
            event.preventDefault();
            open(at, link);
        });
    });

    root.querySelectorAll('[data-gallery-mosaic] img').forEach((img, at) => {
        if (at >= items.length) return;
        img.classList.add('cursor-zoom-in');
        img.addEventListener('click', () => open(at, root.querySelector('[data-gallery-open]')));
    });

    dialog.querySelector('[data-gallery-prev]')?.addEventListener('click', () => show(index - 1));
    dialog.querySelector('[data-gallery-next]')?.addEventListener('click', () => show(index + 1));
    dialog.querySelector('[data-gallery-close]')?.addEventListener('click', () => dialog.close());

    dialog.addEventListener('keydown', (event) => {
        if (event.key === 'ArrowLeft') {
            event.preventDefault();
            show(index + 1);
        } else if (event.key === 'ArrowRight') {
            event.preventDefault();
            show(index - 1);
        }
    });

    // A click on the backdrop (outside the dialog box) closes it.
    dialog.addEventListener('click', (event) => {
        if (event.target === dialog) dialog.close();
    });

    dialog.addEventListener('close', () => {
        stage.replaceChildren();
        opener?.focus?.();
    });
};
