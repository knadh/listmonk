// Shared data, i18n, and DOM helpers for the rich-text editor.
import { I18n } from '../i18n.js';

export const i18n = new I18n((typeof window !== 'undefined' && window._LM_I18N) || {});
export const t = (k) => i18n.t(k);

// Nearest element matching `sel` at the editor's current selection
export function closestInSelection(ed, root, sel) {
  const range = ed.getSelection();
  if (!range) {
    return null;
  }

  let node = range.commonAncestorContainer;
  if (node.nodeType === 3) {
    node = node.parentNode;
  }
  const el = node?.closest(sel);
  if (el && root.contains(el)) {
    return el;
  }

  // Selection wrapping an element.
  const at = range.startContainer.nodeType === 1
    ? range.startContainer.childNodes[range.startOffset] : null;
  if (at?.nodeType !== 1) {
    return null;
  }
  const inner = at.matches(sel) ? at : at.querySelector(sel);
  return inner && root.contains(inner) ? inner : null;
}

// Colour swatches for the fore/highlight colour pickers (applied inline for email).
export const PALETTE = [
  '#BFEDD2', '#FBEEB8', '#F8CAC6', '#C2E0F4', '#2DC26B',
  '#F1C40F', '#E03E2D', '#B96AD9', '#3598DB', '#169179', '#E67E23',
  '#BA372A', '#843FA1', '#236FA1', '#ECF0F1', '#CED4D9', '#95A5A6',
  '#7E8C8D', '#34495E', '#000000', '#ffffff', '#D95757', '#C80707',
];

export const EMOJIS = [
  '😀', '😃', '😄', '😁', '😆', '😅', '😂', '🙂', '😉', '😊', '😍', '😘', '😎', '🤔', '😐', '😴',
  '😮', '😢', '😭', '😡', '👍', '👎', '👌', '🙏', '👏', '💪', '🙌', '👋', '🤝', '✌️', '❤️', '🧡',
  '💛', '💚', '💙', '💜', '🖤', '⭐', '🌟', '✨', '🔥', '💯', '✅', '❌', '⚠️', '❓', '❗', '💡',
  '📧', '📨', '📩', '📢', '🎉', '🎊', '🎁', '🚀', '📈', '📉', '💰', '🛒', '📅', '⏰', '🔔', '🔗',
];

export const CHARS = [
  '©', '®', '™', '€', '£', '¥', '¢', '§', '¶', '†', '‡', '•', '·', '…', '‰', '′',
  '″', '«', '»', '‹', '›', '“', '”', '‘', '’', '–', '—', '¡', '¿', '×', '÷', '±',
  '≠', '≤', '≥', '≈', '∞', '°', 'µ', 'α', 'β', 'γ', 'π', 'Ω', '←', '→', '↑', '↓',
  '¼', '½', '¾', '⅓', '⅔', 'ª', 'º', '№', '☑', '☐', '★', '☆', '♥', '♦', '♣', '♠',
];
