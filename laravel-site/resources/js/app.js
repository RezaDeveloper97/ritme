/**
 * Entry module — intentionally tiny.
 *
 * Page behaviour lives in `resources/js/modules/<name>.js`, each exporting a default
 * `(element) => void`. Markup opts in with `data-module="<name>"` (space-separate several).
 * Only modules present on the page are fetched (Vite splits each into its own hashed chunk),
 * so pages pay only for what they use. Every element is initialised at most once.
 */
const loaders = import.meta.glob('./modules/*.js');

const loaderFor = (name) => loaders[`./modules/${name}.js`];

export function boot(root = document) {
    root.querySelectorAll('[data-module]').forEach((element) => {
        if (element.dataset.moduleBooted !== undefined) {
            return;
        }
        element.dataset.moduleBooted = '';

        element.dataset.module
            .split(/\s+/)
            .filter(Boolean)
            .forEach((name) => {
                const load = loaderFor(name);

                if (!load) {
                    if (import.meta.env.DEV) {
                        console.warn(`[ritme] unknown data-module "${name}"`);
                    }
                    return;
                }

                load()
                    .then((module) => module.default(element))
                    .catch((error) => console.error(`[ritme] data-module "${name}" failed`, error));
            });
    });
}

boot();
