import { z } from 'zod';

import { FONT_FAMILY_NAMES } from './fontFamily';

export function zColor() {
  return z.string().regex(/^#[0-9a-fA-F]{6}$/);
}

export function zFontFamily() {
  return z.enum(FONT_FAMILY_NAMES);
}

export function zFontWeight() {
  return z.enum(['bold', 'normal']);
}

export function zTextAlign() {
  return z.enum(['left', 'center', 'right']);
}

export function zPadding() {
  return z.object({
    top: z.number(),
    bottom: z.number(),
    right: z.number(),
    left: z.number(),
  });
}
