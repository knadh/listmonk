export const models = Object.freeze({
  // This is the config loaded from /api/config.js directly onto the page
  // via a <script> tag.
  serverConfig: 'serverConfig',

  dashboard: 'dashboard',
  lists: 'lists',
  subscribers: 'subscribers',
  campaigns: 'campaigns',
  templates: 'templates',
  media: 'media',
  settings: 'settings',
});

// Ad-hoc URIs that are used outside of vuex requests.
export const uris = Object.freeze({
  previewCampaign: '/api/campaigns/:id/preview',
  previewTemplate: '/api/templates/:id/preview',
  previewRawTemplate: '/api/templates/preview',
});

// Keys used in Vuex store.
export const storeKeys = Object.freeze({
  models: 'models',
  isLoading: 'isLoading',
});

export const timestamp = 'ddd D MMM YYYY, hh:mm A';

export const colors = Object.freeze({
  primary: '#7f2aff',
});
