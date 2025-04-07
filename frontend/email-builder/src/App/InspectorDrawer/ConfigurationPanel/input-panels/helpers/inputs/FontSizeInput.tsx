import React, { useState } from 'react';

import { TextFieldsOutlined } from '@mui/icons-material';
import { InputLabel, Stack } from '@mui/material';

import RawSliderInput from './raw/RawSliderInput';

type Props = {
  label: string;
  defaultValue: number;
  onChange: (v: number) => void;
};
export default function FontSizeInput({ label, defaultValue, onChange }: Props) {
  const [value, setValue] = useState(defaultValue);
  const handleChange = (value: number) => {
    setValue(value);
    onChange(value);
  };
  return (
    <Stack spacing={1} alignItems="flex-start">
      <InputLabel shrink>{label}</InputLabel>
      <RawSliderInput
        iconLabel={<TextFieldsOutlined sx={{ fontSize: 16 }} />}
        value={value}
        setValue={handleChange}
        units="px"
        step={1}
        min={10}
        max={48}
      />
    </Stack>
  );
}
