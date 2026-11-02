import { autocomp } from '@knadh/autocomp';

// Get the key of a tag, either .id for an object or the string itself.
function tagKey(tag) {
  const item = tag.item ?? tag;
  return typeof item === 'string' ? item : item.id;
}

// Render a resource + avatar from objects.
export function tagNode(item, kind, small = false, label = item.name) {
  if (typeof item === 'string') return item;
  const row = document.createElement('span');
  row.className = small ? 'avatar xsmall center' : 'avatar';
  row.item = item;
  row.innerHTML = `<span class="avatar-icon avatar-${kind}"><span class="avatar-initials" aria-hidden="true"></span></span><span class="avatar-label"></span>`;

  // Match the initials used by the server-rendered avatars.
  const words = item.name.match(/[\p{L}\p{Nd}]+/gu) || [];
  const initials = words.length > 1
    ? [...words[0]][0] + [...words[1]][0]
    : [...(words[0] || '?')].slice(0, 2).join('');
  row.querySelector('.avatar-initials').textContent = initials.toUpperCase();
  row.querySelector('.avatar-label').textContent = label;
  return row;
}

// Attach string or object suggestions to Oat taginput.
export function tagAutocomplete(el, { query, render = (item) => item, debounce = 0 }) {
  const ti = el.closest('ot-taginput');
  autocomp(el, {
    debounce,
    async onQuery(q) {
      const results = await query(q);
      const chosen = new Set(ti.value.map(tagKey));
      return results.filter((item) => !chosen.has(tagKey(item))).slice(0, 10);
    },

    onRender(item) {
      const node = render(item, false);
      return typeof node === 'string' ? document.createTextNode(node) : node;
    },

    onSelect(item) {
      if (!ti.value.some((tag) => tagKey(tag) === tagKey(item))) {
        ti.add(render(item, true));
      }
      return '';
    },
  });

  // Show suggestions before the user starts typing.
  ['focus', 'click'].forEach((event) => {
    el.addEventListener(event, () => el.dispatchEvent(new Event('input')));
  });
}
