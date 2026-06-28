import React from 'react';

import { FormControlLabel, Switch } from '@mui/material';

type Props = {
  label: string;
  defaultValue: boolean;
  onChange: (value: boolean) => void;
};

export default function BooleanInput({ label, defaultValue, onChange }: Props) {
  return (
    <FormControlLabel
      label={label}
      control={
        <Switch
          checked={defaultValue}
          onChange={(_, checked: boolean) => {
            onChange(checked);
          }}
        />
      }
    />
  );
}
