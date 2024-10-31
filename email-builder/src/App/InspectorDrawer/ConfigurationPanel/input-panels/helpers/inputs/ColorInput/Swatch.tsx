import React from 'react';

import { Box, Button, SxProps } from '@mui/material';

type Props = {
  paletteColors: string[];
  value: string;
  onChange: (value: string) => void;
};

const TILE_BUTTON: SxProps = {
  width: 24,
  height: 24,
};
export default function Swatch({ paletteColors, value, onChange }: Props) {
  const renderButton = (colorValue: string) => {
    return (
      <Button
        key={colorValue}
        onClick={() => onChange(colorValue)}
        sx={{
          ...TILE_BUTTON,
          backgroundColor: colorValue,
          border: '1px solid',
          borderColor: value === colorValue ? 'black' : 'grey.200',
          minWidth: 24,
          display: 'inline-flex',
          '&:hover': {
            backgroundColor: colorValue,
            borderColor: 'grey.500',
          },
        }}
      />
    );
  };
  return (
    <Box width="100%" sx={{ display: 'grid', gap: 1, gridTemplateColumns: '1fr 1fr 1fr 1fr 1fr 1fr' }}>
      {paletteColors.map((c) => renderButton(c))}
    </Box>
  );
}
