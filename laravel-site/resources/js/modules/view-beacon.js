/**
 * data-module="view-beacon" on the article meta line (components/blog/meta): counts one view per post per tab
 * session, also when the page came from the full-page cache. Sends a same-origin `navigator.sendBeacon` POST
 * (fallback: keepalive fetch without credentials) to data-view-url with the post id; the route has no session,
 * cookies or CSRF and the server ignores bots and Save-Data on its own as well.
 *
 * Skipped: automation (navigator.webdriver), Save-Data, prerender, an already counted post in this tab. A page
 * opened in a background tab is counted only once it becomes visible.
 */
const KEY = 'rt:viewed:';

const seen = (id) => {
    try {
        return window.sessionStorage.getItem(KEY + id) === '1';
    } catch {
        return false;
    }
};

const remember = (id) => {
    try {
        window.sessionStorage.setItem(KEY + id, '1');
    } catch {
        // storage blocked: the beacon is still sent once for this page view
    }
};

const send = (url, id) => {
    const body = new FormData();
    body.append('id', id);
    if (typeof navigator.sendBeacon === 'function' && navigator.sendBeacon(url, body)) return;
    if (typeof fetch === 'function') {
        fetch(url, { method: 'POST', body, keepalive: true, credentials: 'omit', mode: 'same-origin' }).catch(() => {});
    }
};

export default (element) => {
    const url = element.dataset.viewUrl;
    const id = element.dataset.viewId;
    if (!url || !id || navigator.webdriver) return;
    if (navigator.connection && navigator.connection.saveData) return;
    if (seen(id)) return;

    const fire = () => {
        if (seen(id)) return;
        remember(id);
        send(url, id);
    };

    if (document.visibilityState === 'visible') {
        fire();
        return;
    }
    const onVisible = () => {
        if (document.visibilityState !== 'visible') return;
        document.removeEventListener('visibilitychange', onVisible);
        fire();
    };
    document.addEventListener('visibilitychange', onVisible);
};
