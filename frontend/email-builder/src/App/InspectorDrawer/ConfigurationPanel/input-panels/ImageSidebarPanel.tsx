import {
  VerticalAlignBottomOutlined,
  VerticalAlignCenterOutlined,
  VerticalAlignTopOutlined,
} from '@mui/icons-material';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import { Checkbox, FormControlLabel, Stack, ToggleButton } from '@mui/material';
import { ImageProps, ImagePropsSchema } from '@usewaypoint/block-image';
import React, { useEffect, useRef, useState } from 'react';
import { z } from 'zod';

import BaseSidebarPanel from './helpers/BaseSidebarPanel';
import RadioGroupInput from './helpers/inputs/RadioGroupInput';
import TextDimensionInput from './helpers/inputs/TextDimensionInput';
import TextInput from './helpers/inputs/TextInput';
import MultiStylePropertyPanel from './helpers/style-inputs/MultiStylePropertyPanel';

// would strip embed/uuid during validation.
const ImgPropsSchema = ImagePropsSchema.extend({
  props: z.object({
    width: z.number().nullable().optional(),
    height: z.number().nullable().optional(),
    url: z.string().nullable().optional(),
    alt: z.string().nullable().optional(),
    linkHref: z.string().nullable().optional(),
    contentAlignment: z.enum(['top', 'middle', 'bottom']).nullable().optional(),
    embed: z.boolean().nullable().optional(),
    uuid: z.string().nullable().optional(),
  }).nullable().optional(),
});
type ListmonkImageProps = z.infer<typeof ImgPropsSchema>;

type ImageSidebarPanelProps = {
  data: ImageProps;
  setData: (v: ImageProps) => void;
};
export default function ImageSidebarPanel({ data, setData }: ImageSidebarPanelProps) {
  const [, setErrors] = useState<Zod.ZodError | null>(null);

  const dataRef = useRef<ListmonkImageProps>(data as ListmonkImageProps);
  dataRef.current = data as ListmonkImageProps;

  const updateData = (d: unknown) => {
    const res = ImgPropsSchema.safeParse(d);
    if (res.success) {
      setData(res.data as ImageProps);
      setErrors(null);
    } else {
      setErrors(res.error);
    }
  };

  useEffect(() => {
    const onMessage = (e: MessageEvent) => {
      if (!e.data || e.data.action !== 'visualeditor.media-uuid') {
        return;
      }
      const cur = dataRef.current;
      const curURL = (cur && cur.props && cur.props.url) || '';
      // Guard against stale UUID deliveries after the user switched blocks.
      if (e.data.url && e.data.url !== curURL) {
        return;
      }
      updateData({ ...cur, props: { ...(cur && cur.props), uuid: e.data.uuid } });
    };
    window.addEventListener('message', onMessage);
    return () => window.removeEventListener('message', onMessage);
  }, []);

  const props = (data && (data as ListmonkImageProps).props) || {};

  return (
    <BaseSidebarPanel title="Image block">
      <TextInput
        label="Source URL"
        className="image-url"
        defaultValue={data.props?.url ?? ''}
        onChange={(v) => {
          const url = v.trim().length === 0 ? null : v.trim();
          updateData({ ...data, props: { ...data.props, url } });
        }}
      />
      <a href="#" class="select-media"
        style={{ display: 'inline-flex', alignItems: 'center', gap: '0.5rem', marginTop: '5px' }}
        onClick={(e) => {
        // @ts-ignore
        window.parent.postMessage('visualeditor.select-media', '*');
        e.preventDefault();
      }}><CloudUploadIcon style={{fontSize: '1rem'}} /> Select media</a>

      <TextInput
        label="Alt text"
        defaultValue={data.props?.alt ?? ''}
        onChange={(alt) => updateData({ ...data, props: { ...data.props, alt } })}
      />
      <TextInput
        label="Click through URL"
        defaultValue={data.props?.linkHref ?? ''}
        onChange={(v) => {
          const linkHref = v.trim().length === 0 ? null : v.trim();
          updateData({ ...data, props: { ...data.props, linkHref } });
        }}
      />
      <Stack direction="row" spacing={2}>
        <TextDimensionInput
          label="Width"
          defaultValue={data.props?.width}
          onChange={(width) => updateData({ ...data, props: { ...data.props, width } })}
        />
        <TextDimensionInput
          label="Height"
          defaultValue={data.props?.height}
          onChange={(height) => updateData({ ...data, props: { ...data.props, height } })}
        />
      </Stack>

      <RadioGroupInput
        label="Alignment"
        defaultValue={data.props?.contentAlignment ?? 'middle'}
        onChange={(contentAlignment) => updateData({ ...data, props: { ...data.props, contentAlignment } })}
      >
        <ToggleButton value="top">
          <VerticalAlignTopOutlined fontSize="small" />
        </ToggleButton>
        <ToggleButton value="middle">
          <VerticalAlignCenterOutlined fontSize="small" />
        </ToggleButton>
        <ToggleButton value="bottom">
          <VerticalAlignBottomOutlined fontSize="small" />
        </ToggleButton>
      </RadioGroupInput>

      <FormControlLabel
        control={
          <Checkbox
            size="small"
            checked={Boolean(props.embed)}
            onChange={(e) => updateData({ ...data, props: { ...data.props, embed: e.target.checked } })}
          />
        }
        label="Embed inline (CID)"
      />

      <MultiStylePropertyPanel
        names={['backgroundColor', 'textAlign', 'padding']}
        value={data.style}
        onChange={(style) => updateData({ ...data, style })}
      />
    </BaseSidebarPanel>
  );
}
