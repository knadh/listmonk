import React from 'react';

import { ExpandMore } from '@mui/icons-material';
import {
  Accordion, AccordionDetails, AccordionSummary, Stack, Typography,
} from '@mui/material';

import { MARKDOWN_ELEMENTS } from '../../../../../documents/blocks/Text/markdownStyles';

import TextInput from './inputs/TextInput';

type Props = {
  value: Record<string, string> | null | undefined;
  onChange: (v: Record<string, string> | null) => void;
};

export default function MarkdownStylesInput({ value, onChange }: Props) {
  const setElementStyle = (key: string, style: string) => {
    const next = { ...(value ?? {}) };
    if (style.trim()) {
      next[key] = style;
    } else {
      delete next[key];
    }
    onChange(Object.keys(next).length ? next : null);
  };

  return (
    <Accordion disableGutters variant="outlined">
      <AccordionSummary expandIcon={<ExpandMore />}>
        <Typography variant="body2">Markdown element styles</Typography>
      </AccordionSummary>
      <AccordionDetails>
        <Stack spacing={2}>
          <Typography variant="caption" color="text.secondary">
            Inline CSS added to each Markdown element, for example: margin: 0 0 8px; color: #333333
          </Typography>
          {MARKDOWN_ELEMENTS.map(({ key, label }) => (
            <TextInput
              key={key}
              label={label}
              placeholder="margin: 0 0 8px"
              defaultValue={value?.[key] ?? ''}
              onChange={(v) => setElementStyle(key, v)}
            />
          ))}
        </Stack>
      </AccordionDetails>
    </Accordion>
  );
}
