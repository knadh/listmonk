import React, { useState } from 'react';

import { ToggleButton } from '@mui/material';

import RadioGroupInput from './RadioGroupInput';

type Props = {
  label: string;
  defaultValue: string;
  onChange: (value: string) => void;
};
export default function FontWeightInput({ label, defaultValue, onChange }: Props) {
  const [value, setValue] = useState(defaultValue);
  return (
    <RadioGroupInput
      label={label}
      defaultValue={value}
      onChange={(fontWeight) => {
        setValue(fontWeight);
        onChange(fontWeight);
      }}
    >
      <ToggleButton value="normal">Regular</ToggleButton>
      <ToggleButton value="bold">Bold</ToggleButton>
    </RadioGroupInput>
  );
}
