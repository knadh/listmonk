<template>
  <span :class="badgeClass" :data-variant="variant || null">
    <slot />
  </span>
</template>

<script>
const variantMap = {
  active: 'success',
  api: 'secondary',
  blocklisted: 'danger',
  campaign: 'secondary',
  campaign_visual: 'secondary',
  disabled: 'danger',
  done: 'success',
  double: 'warning',
  draft: 'secondary',
  enabled: 'success',
  error: 'danger',
  failed: 'danger',
  optin: 'warning',
  paused: 'warning',
  primary: 'secondary',
  private: 'secondary',
  public: 'success',
  running: 'warning',
  scheduled: 'warning',
  single: 'secondary',
  status_unconfirmed: 'warning',
  status_unsubscribed: 'danger',
  status_confirmed: 'success',
};

export default {
  name: 'OatBadge',
  props: {
    type: {
      type: String,
      default: '',
    },
    variant: {
      type: String,
      default: '',
    },
  },
  computed: {
    badgeClass() {
      const typeClass = (this.type || '').trim();
      const key = typeClass.replace(/-/g, '_');
      return ['badge', typeClass, key !== typeClass ? key : null, this.variant || variantMap[key] || 'outline'];
    },
  },
};
</script>
