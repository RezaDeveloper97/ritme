import StarterKit from '@tiptap/starter-kit';

/** The editor's schema — only what the backend sanitizer keeps (see rich-text.ts). */
export const editorExtensions = [
  StarterKit.configure({
    heading: { levels: [2, 3] },
    link: {
      openOnClick: false,
      autolink: true,
      defaultProtocol: 'https',
      protocols: ['http', 'https', 'mailto', 'tel'],
      // No `class`: the sanitizer would strip it; keep editor output = saved HTML.
      HTMLAttributes: { class: null, rel: 'noopener noreferrer nofollow', target: '_blank' },
    },
    // Code blocks add `class="language-…"`; plain <pre><code> is enough for content.
    codeBlock: { HTMLAttributes: { class: null } },
  }),
];
