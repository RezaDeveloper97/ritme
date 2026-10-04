/**
 * data-module="menu" on the header burger: toggles the element named by aria-controls (`hidden` + aria-expanded),
 * swaps the button label, closes on Esc (focus back to the button), on a link click and above the 1024px breakpoint.
 */
export default (button) => {
    const menu = document.getElementById(button.getAttribute('aria-controls'));
    if (!menu) return;
    const labels = [button.getAttribute('aria-label'), button.dataset.labelClose];
    const set = (open, focus) => {
        button.setAttribute('aria-expanded', String(open));
        button.setAttribute('aria-label', labels[+open] || labels[0]);
        menu.hidden = !open;
        if (focus) button.focus();
    };
    button.addEventListener('click', () => set(menu.hidden));
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && !menu.hidden) set(false, true);
    });
    menu.addEventListener('click', (e) => {
        if (e.target.closest('a')) set(false);
    });
    matchMedia('(min-width: 1025px)').addEventListener('change', (e) => {
        if (e.matches) set(false);
    });
};
