#!/usr/bin/env python3
"""Validate bundled translations and source references."""
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent
PARAM = re.compile(r"(?<!\{)\{([\w.-]+)\}(?!\})")
CALL = re.compile(
    r'''(?:\.T[sc]?\(\s*|\.T[sc]?\s+|\bi18n\.[tT][sc]?\(\s*|\bt\(\s*)["']([\w.]+)["']'''
)
JS_CALL = re.compile(r'''(?:\bi18n\.(?:t|ts|tc)|\bt)\(\s*['"]([\w.]+)['"]''')
NESTED = re.compile(r'\{((?:globals|campaigns|subscribers|settings|users|lists|media)\.[\w.]+)\}')
DYNAMIC = {
    'globals.terms.': 'enabled disabled blocklisted subscribed unsubscribed confirmed unconfirmed cancelled draft finished paused running scheduled public private'.split(),
    'bounces.': 'hard soft complaint'.split(),
    'editor.contentType.': 'richtext html markdown plain visual'.split(),
    'editor.': 'rowAbove rowBelow rowDelete colLeft colRight colDelete tableDelete'.split(),
}

# Stored default email templates still reference this legacy key.
SKIP_LIST = {'email.unsub': 'globals.buttons.unsubscribe'}
errors = []


def unique_object(pairs):
    out = {}
    for key, value in pairs:
        if key in out:
            raise ValueError(f'duplicate key: {key}')
        out[key] = value
    return out


def read_json(path):
    return json.loads(path.read_text(), object_pairs_hook=unique_object)


def main():
    base = read_json(ROOT / 'i18n/en.json')
    key_pattern = re.compile(r'(?<![\w.])(?:' + '|'.join(re.escape(k) for k in base) + r')(?![\w.])')
    locales = sorted((ROOT / 'i18n').glob('*.json'))
    for path in locales:
        try:
            data = read_json(path)
        except (ValueError, UnicodeError) as exc:
            errors.append(f'{path.name}: {exc}')
            continue

        if data.keys() != base.keys():
            errors.append(f'{path.name}: missing={sorted(base.keys() - data.keys())}, extra={sorted(data.keys() - base.keys())}')

        if data.get('_.code') != path.stem:
            errors.append(f'{path.name}: incorrect locale code')

        for legacy, canonical in SKIP_LIST.items():
            if legacy not in data or data[legacy] != data.get(canonical):
                errors.append(f'{path.name}: compatibility key {legacy} must match {canonical}')

        for key in data.keys() & base.keys():
            value = data[key]
            if not isinstance(value, str) or not value.strip():
                errors.append(f'{path.name}: empty/non-string {key}')
                continue

            expected = set(PARAM.findall(base[key]))

            # The runtime selects each branch separately. Every branch must
            # retain the parameters, even when plural rules need neutral wording.
            for branch in value.split('|'):
                if set(PARAM.findall(branch)) != expected:
                    errors.append(f'{path.name}: placeholder mismatch in {key}')

            if '|' in value and '|' not in base[key]:
                errors.append(f'{path.name}: unexpected plural branches in {key}')

    lines = (ROOT / 'static/admin/i18n.txt').read_text().splitlines()
    exported = [line.strip() for line in lines if line.strip() and not line.startswith('#')]

    if len(exported) != len(set(exported)):
        errors.append('Duplicate browser translation exports')
    used = set(exported) | set(SKIP_LIST)
    browser_used = set()

    for prefix, suffixes in DYNAMIC.items():
        used.update(prefix + suffix for suffix in suffixes)
    browser_used.update('globals.terms.' + status for status in 'cancelled draft finished paused running scheduled'.split())

    for prefix in ('editor.', 'editor.contentType.'):
        browser_used.update(prefix + suffix for suffix in DYNAMIC[prefix])

    files = subprocess.check_output(['git', 'ls-files'], cwd=ROOT, text=True).splitlines()
    source_count = 0
    for name in files:
        if not name.startswith(('cmd/', 'internal/', 'static/')):
            continue

        # Third-party bundles are not listmonk translation consumers.
        if '/assets/static/' in name or '/vendor/' in name:
            continue

        path = ROOT / name
        if path.suffix not in ('.go', '.js', '.html', '.tpl', '.txt', '.json'):
            continue

        source_count += 1
        text = path.read_text()
        if path.suffix == '.json':
            def strings(value):
                if isinstance(value, str):
                    yield value
                elif isinstance(value, dict):
                    for item in value.values():
                        yield from strings(item)
                elif isinstance(value, list):
                    for item in value:
                        yield from strings(item)
            text = '\n'.join(strings(read_json(path)))
        refs = set(CALL.findall(text)) | set(NESTED.findall(text))

        for key in refs:
            if key not in base:
                errors.append(f'{name}: unknown translation {key}')
        used.update(refs)

        # Also include keys stored as sort-label strings, dict values, etc.
        used.update(key_pattern.findall(text))
        if name.startswith('static/admin/assets/js/'):
            browser_used.update(JS_CALL.findall(text))

    for key in used - base.keys():
        errors.append(f'Unknown exported/dynamic key: {key}')

    for key in browser_used - set(exported):
        errors.append(f'Browser key is not exported: {key}')

    unused = sorted(base.keys() - used - {'_.code', '_.name'})

    if errors:
        print('\n'.join(errors), file=sys.stderr)
        return 1

    print(f'OK: {len(locales)} locales, {len(base)} keys each; {source_count} source files; {len(exported)} browser exports.')

    if unused:
        print('check unused keys (not automatically removed): ' + ', '.join(unused))
    return 0


if __name__ == '__main__':
    sys.exit(main())
