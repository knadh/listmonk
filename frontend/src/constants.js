export const models = Object.freeze({
  serverConfig: 'serverConfig',
  lang: 'lang',
  dashboard: 'dashboard',
  lists: 'lists',
  subscribers: 'subscribers',
  campaigns: 'campaigns',
  templates: 'templates',
  media: 'media',
  bounces: 'bounces',
  settings: 'settings',
  logs: 'logs',
  maintenance: 'maintenance',
});

// Ad-hoc URIs that are used outside of vuex requests.
const rootURL = process.env.VUE_APP_ROOT_URL || '/';
const baseURL = process.env.BASE_URL.replace(/\/$/, '');

export const uris = Object.freeze({
  previewCampaign: '/api/campaigns/:id/preview',
  previewTemplate: '/api/templates/:id/preview',
  previewRawTemplate: '/api/templates/preview',
  exportSubscribers: '/api/subscribers/export',
  base: `${baseURL}/static`,
  root: rootURL,
  static: `${baseURL}/static`,
});


// Keys used in Vuex store.
export const storeKeys = Object.freeze({
  models: 'models',
  isLoading: 'isLoading',
});

export const timestamp = 'ddd D MMM YYYY, hh:mm A';

export const colors = Object.freeze({
  primary: '#0055d4',
});

export const regDuration = '[0-9]+(ms|s|m|h|d)';
