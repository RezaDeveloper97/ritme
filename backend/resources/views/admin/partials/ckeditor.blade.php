{{--
    Rich-text editor bootstrap (CKEditor 4.22.1 LGPL, vendored in
    public/assets/ckeditor and served through /admin/ckeditor/*).

    Included once per page by <x-admin.bilingual type="editor" />. Every
    textarea carrying data-rich-editor is upgraded on DOMContentLoaded, so
    fields rendered after this script still get an editor.
--}}
<script src="{{ route('admin.ckeditor', ['path' => 'ckeditor.js']) }}"></script>
<script>
    document.addEventListener('DOMContentLoaded', function () {
        if (typeof CKEDITOR === 'undefined') {
            return; // Editor unavailable — the plain textareas stay usable.
        }

        // No outbound version ping: the server has no international network.
        CKEDITOR.config.versionCheck = false;

        // Only buttons the bundled "standard" build actually ships — anything
        // else (Justify, Bidi, …) belongs to the full build and would be
        // silently dropped from the toolbar.
        var toolbar = [
            { name: 'clipboard', items: ['Undo', 'Redo', '-', 'PasteText', 'PasteFromWord'] },
            { name: 'styles', items: ['Format'] },
            { name: 'basicstyles', items: ['Bold', 'Italic', 'Underline', 'Strike', 'RemoveFormat'] },
            { name: 'paragraph', items: ['NumberedList', 'BulletedList', 'Outdent', 'Indent', 'Blockquote'] },
            { name: 'links', items: ['Link', 'Unlink'] },
            { name: 'insert', items: ['Table', 'HorizontalRule', 'SpecialChar'] },
            { name: 'tools', items: ['Maximize', 'Source'] },
        ];

        document.querySelectorAll('textarea[data-rich-editor]').forEach(function (textarea) {
            var rtl = textarea.getAttribute('dir') !== 'ltr';

            CKEDITOR.replace(textarea, {
                customConfig: '',
                versionCheck: false,
                language: textarea.getAttribute('data-editor-language') || 'fa',
                contentsLangDirection: rtl ? 'rtl' : 'ltr',
                // Keep Persian characters as themselves instead of &#1575; entities.
                entities: false,
                entities_latin: false,
                height: 220,
                toolbar: toolbar,
                format_tags: 'p;h2;h3;h4',
                // scayt/wsc talk to WebSpellChecker's servers, which are
                // unreachable from the production host — drop them entirely.
                removePlugins: 'elementspath,scayt,wsc',
                removeDialogTabs: 'link:advanced;table:advanced',
                // No allowedContent override: content is filtered to what these
                // buttons can produce, which drops scripts and inline handlers.
            });
        });
    });
</script>
