import React, { useState } from 'react';

import { FormControlLabel, Switch } from '@mui/material';

type Props = {
  label: string;
  defaultValue: boolean;
  onChange: (value: boolean) => void;
};

export default function BooleanInput({ label, defaultValue, onChange }: Props) {
  const [value, setValue] = useState(defaultValue);
  return (
    <FormControlLabel
      label={label}
      control={
        <Switch
          checked={value}
          onChange={(_, checked: boolean) => {
            setValue(checked);
            onChange(checked);
          }}
        />
      }
    />
  );
}
