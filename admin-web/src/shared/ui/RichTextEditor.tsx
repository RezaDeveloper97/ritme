'use client';

import { EditorContent, useEditor, useEditorState, type Editor } from '@tiptap/react';
import { useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { Icon, type IconName } from './Icon';
import { editorExtensions } from './editor-extensions';
import { normalizeEditorHtml } from './rich-text';

/**
 * TipTap editor for admin-authored HTML (articles, info sections, …).
 *
 * The extension set is limited to what the backend sanitizer keeps
 * (backend-go/internal/content/sanitizer: p, h2/h3, strong, em, u, s, ul/ol/li,
 * blockquote, pre/code, hr, br, a[href|title|target|rel]) so what the admin sees
 * is what the app renders. Anything else pasted in is dropped by the editor
 * schema first and by the sanitizer again on save.
 *
 * Emits '' for an empty document (not '<p></p>') so "required" works.
 */
export function RichTextEditor({
  value,
  onChange,
  dir,
  id,
  invalid,
  ariaLabel,
  ariaDescribedBy,
}: {
  value: string;
  onChange: (html: string) => void;
  dir?: 'rtl' | 'ltr';
  id?: string;
  invalid?: boolean;
  ariaLabel?: string;
  ariaDescribedBy?: string;
}) {
  // useEditor captures its options once; read the latest onChange through a ref.
  const onChangeRef = useRef(onChange);
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  const editor = useEditor({
    immediatelyRender: false,
    extensions: editorExtensions,
    content: value,
    editorProps: {
      attributes: {
        class: 'prose-admin',
        ...(dir ? { dir } : {}),
        ...(id ? { id } : {}),
        ...(ariaLabel ? { 'aria-label': ariaLabel } : {}),
        ...(ariaDescribedBy ? { 'aria-describedby': ariaDescribedBy } : {}),
        role: 'textbox',
        'aria-multiline': 'true',
      },
    },
    onUpdate: ({ editor: e }) => onChangeRef.current(normalizeEditorHtml(e.isEmpty ? '' : e.getHTML())),
  });

  // Follow outside changes (record loaded, form reset) without echoing them back.
  useEffect(() => {
    if (!editor) return;
    const current = normalizeEditorHtml(editor.isEmpty ? '' : editor.getHTML());
    if (current !== value) editor.commands.setContent(value || '', { emitUpdate: false });
  }, [editor, value]);

  return (
    <div className="rte" aria-invalid={invalid || undefined}>
      {editor ? <Toolbar editor={editor} /> : <div className="rte-toolbar h-[2.55rem]" />}
      <EditorContent editor={editor} className="rte-content" />
    </div>
  );
}

function Toolbar({ editor }: { editor: Editor }) {
  const t = useTranslations('editor');
  const state = useEditorState({
    editor,
    selector: ({ editor: e }) => ({
      bold: e.isActive('bold'),
      italic: e.isActive('italic'),
      underline: e.isActive('underline'),
      strike: e.isActive('strike'),
      h2: e.isActive('heading', { level: 2 }),
      h3: e.isActive('heading', { level: 3 }),
      bulletList: e.isActive('bulletList'),
      orderedList: e.isActive('orderedList'),
      blockquote: e.isActive('blockquote'),
      link: e.isActive('link'),
      canUndo: e.can().undo(),
      canRedo: e.can().redo(),
    }),
  });

  const chain = () => editor.chain().focus();
  const setLink = () => {
    const previous = (editor.getAttributes('link').href as string | undefined) ?? '';
    const href = window.prompt(t('linkPrompt'), previous);
    if (href === null) return;
    if (href.trim() === '') chain().extendMarkRange('link').unsetLink().run();
    else chain().extendMarkRange('link').setLink({ href: href.trim() }).run();
  };

  type Item =
    | { kind: 'text'; key: string; label: string; glyph: string; active: boolean; run: () => void }
    | { kind: 'icon'; key: string; label: string; icon: IconName; active?: boolean; disabled?: boolean; run: () => void }
    | { kind: 'sep'; key: string };

  const items: Item[] = [
    { kind: 'text', key: 'b', label: t('bold'), glyph: 'B', active: state.bold, run: () => chain().toggleBold().run() },
    { kind: 'text', key: 'i', label: t('italic'), glyph: 'I', active: state.italic, run: () => chain().toggleItalic().run() },
    { kind: 'text', key: 'u', label: t('underline'), glyph: 'U', active: state.underline, run: () => chain().toggleUnderline().run() },
    { kind: 'text', key: 's', label: t('strike'), glyph: 'S', active: state.strike, run: () => chain().toggleStrike().run() },
    { kind: 'sep', key: 'sep1' },
    { kind: 'text', key: 'h2', label: t('h2'), glyph: 'H2', active: state.h2, run: () => chain().toggleHeading({ level: 2 }).run() },
    { kind: 'text', key: 'h3', label: t('h3'), glyph: 'H3', active: state.h3, run: () => chain().toggleHeading({ level: 3 }).run() },
    { kind: 'icon', key: 'ul', label: t('bulletList'), icon: 'listBullet', active: state.bulletList, run: () => chain().toggleBulletList().run() },
    { kind: 'icon', key: 'ol', label: t('orderedList'), icon: 'listOrdered', active: state.orderedList, run: () => chain().toggleOrderedList().run() },
    { kind: 'icon', key: 'q', label: t('blockquote'), icon: 'quote', active: state.blockquote, run: () => chain().toggleBlockquote().run() },
    { kind: 'icon', key: 'a', label: t('link'), icon: 'link', active: state.link, run: setLink },
    { kind: 'icon', key: 'hr', label: t('hr'), icon: 'divider', run: () => chain().setHorizontalRule().run() },
    { kind: 'sep', key: 'sep2' },
    { kind: 'icon', key: 'undo', label: t('undo'), icon: 'undo', disabled: !state.canUndo, run: () => chain().undo().run() },
    { kind: 'icon', key: 'redo', label: t('redo'), icon: 'redo', disabled: !state.canRedo, run: () => chain().redo().run() },
  ];

  return (
    <div className="rte-toolbar" role="toolbar" aria-label={t('toolbar')}>
      {items.map((item) =>
        item.kind === 'sep' ? (
          <span key={item.key} className="rte-sep" aria-hidden="true" />
        ) : (
          <button
            key={item.key}
            type="button"
            className="rte-btn"
            title={item.label}
            aria-label={item.label}
            aria-pressed={item.kind === 'text' ? item.active : item.active === undefined ? undefined : item.active}
            disabled={item.kind === 'icon' ? item.disabled : undefined}
            onMouseDown={(e) => e.preventDefault()}
            onClick={item.run}
          >
            {item.kind === 'text' ? (
              item.glyph
            ) : (
              <Icon name={item.icon} size={16} className={item.key === 'undo' || item.key === 'redo' ? 'rtl:-scale-x-100' : undefined} />
            )}
          </button>
        ),
      )}
    </div>
  );
}
