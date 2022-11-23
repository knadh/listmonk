import { ToastProgrammatic as Toast } from 'buefy';
import axios from 'axios';
import qs from 'qs';
import store from '../store';
import { models } from '../constants';
import Utils from '../utils';

const http = axios.create({
  baseURL: process.env.VUE_APP_ROOT_URL || '/',
  withCredentials: false,
  responseType: 'json',

  // Override the default serializer to switch params from becoming []id=a&[]id=b ...
  // in GET and DELETE requests to id=a&id=b.
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});

const utils = new Utils();

// Intercept requests to set the 'loading' state of a model.
http.interceptors.request.use((config) => {
  if ('loading' in config) {
    store.commit('setLoading', { model: config.loading, status: true });
  }
  return config;
}, (error) => Promise.reject(error));

// Intercept responses to set them to store.
http.interceptors.response.use((resp) => {
  // Clear the loading state for a model.
  if ('loading' in resp.config) {
    store.commit('setLoading', { model: resp.config.loading, status: false });
  }


  let data = {};
  if (typeof resp.data.data === 'object') {
    if (resp.data.data.constructor === Object) {
      data = { ...resp.data.data };
    } else {
      data = [...resp.data.data];
    }

    // Transform keys to camelCase.
    switch (typeof resp.config.camelCase) {
      case 'function':
        data = utils.camelKeys(data, resp.config.camelCase);
        break;
      case 'boolean':
        if (resp.config.camelCase) {
          data = utils.camelKeys(data);
        }
        break;
      default:
        data = utils.camelKeys(data);
        break;
    }
  } else {
    data = resp.data.data;
  }

  // Store the API response for a model.
  if ('store' in resp.config) {
    store.commit('setModelResponse', { model: resp.config.store, data });
  }

  return data;
}, (err) => {
  // Clear the loading state for a model.
  if ('loading' in err.config) {
    store.commit('setLoading', { model: err.config.loading, status: false });
  }

  let msg = '';
  if (err.response.data && err.response.data.message) {
    msg = err.response.data.message;
  } else {
    msg = err.toString();
  }

  if (!err.config.disableToast) {
    Toast.open({
      message: msg,
      type: 'is-danger',
      queue: false,
      position: 'is-top',
      pauseOnHover: true,
    });
  }

  return Promise.reject(err);
});

// API calls accept the following config keys.
// loading: modelName (set's the loading status in the global store: eg: store.loading.lists = true)
// store: modelName (set's the API response in the global store. eg: store.lists: { ... } )

// Health check endpoint that does not throw a toast.
export const getHealth = () => http.get('/api/health',
  { disableToast: true });

export const reloadApp = () => http.post('/api/admin/reload');

// Dashboard
export const getDashboardCounts = () => http.get('/api/dashboard/counts',
  { loading: models.dashboard });

export const getDashboardCharts = () => http.get('/api/dashboard/charts',
  { loading: models.dashboard });

// Lists.
export const getLists = (params) => http.get('/api/lists',
  {
    params: (!params ? { per_page: 'all' } : params),
    loading: models.lists,
    store: models.lists,
  });

export const getList = async (id) => http.get(`/api/lists/${id}`,
  { loading: models.list });

export const createList = (data) => http.post('/api/lists', data,
  { loading: models.lists });

export const updateList = (data) => http.put(`/api/lists/${data.id}`, data,
  { loading: models.lists });

export const deleteList = (id) => http.delete(`/api/lists/${id}`,
  { loading: models.lists });

// Subscribers.
export const getSubscribers = async (params) => http.get('/api/subscribers',
  {
    params,
    loading: models.subscribers,
    store: models.subscribers,
    camelCase: (keyPath) => !keyPath.startsWith('.results.*.attribs'),
  });

export const getSubscriber = async (id) => http.get(`/api/subscribers/${id}`,
  { loading: models.subscribers });

export const getSubscriberBounces = async (id) => http.get(`/api/subscribers/${id}/bounces`,
  { loading: models.bounces });

export const deleteSubscriberBounces = async (id) => http.delete(`/api/subscribers/${id}/bounces`,
  { loading: models.bounces });

export const deleteBounce = async (id) => http.delete(`/api/bounces/${id}`,
  { loading: models.bounces });

export const deleteBounces = async (params) => http.delete('/api/bounces',
  { params, loading: models.bounces });

export const createSubscriber = (data) => http.post('/api/subscribers', data,
  { loading: models.subscribers });

export const updateSubscriber = (data) => http.put(`/api/subscribers/${data.id}`, data,
  { loading: models.subscribers });

export const sendSubscriberOptin = (id) => http.post(`/api/subscribers/${id}/optin`, {},
  { loading: models.subscribers });

export const deleteSubscriber = (id) => http.delete(`/api/subscribers/${id}`,
  { loading: models.subscribers });

export const addSubscribersToLists = (data) => http.put('/api/subscribers/lists', data,
  { loading: models.subscribers });

export const addSubscribersToListsByQuery = (data) => http.put('/api/subscribers/query/lists',
  data, { loading: models.subscribers });

export const blocklistSubscribers = (data) => http.put('/api/subscribers/blocklist', data,
  { loading: models.subscribers });

export const blocklistSubscribersByQuery = (data) => http.put('/api/subscribers/query/blocklist', data,
  { loading: models.subscribers });

export const deleteSubscribers = (params) => http.delete('/api/subscribers',
  { params, loading: models.subscribers });

export const deleteSubscribersByQuery = (data) => http.post('/api/subscribers/query/delete', data,
  { loading: models.subscribers });

// Subscriber import.
export const importSubscribers = (data) => http.post('/api/import/subscribers', data);

export const getImportStatus = () => http.get('/api/import/subscribers');

export const getImportLogs = async () => http.get('/api/import/subscribers/logs',
  { camelCase: false });

export const stopImport = () => http.delete('/api/import/subscribers');

// Bounces.
export const getBounces = async (params) => http.get('/api/bounces',
  { params, loading: models.bounces });

// Campaigns.
export const getCampaigns = async (params) => http.get('/api/campaigns', {
  params,
  loading: models.campaigns,
  store: models.campaigns,
  camelCase: (keyPath) => !keyPath.startsWith('.results.*.headers'),
});

export const getCampaign = async (id) => http.get(`/api/campaigns/${id}`, {
  loading: models.campaigns,
  camelCase: (keyPath) => !keyPath.startsWith('.headers'),
});

export const getCampaignStats = async () => http.get('/api/campaigns/running/stats', {});

export const createCampaign = async (data) => http.post('/api/campaigns', data,
  { loading: models.campaigns });

export const getCampaignViewCounts = async (params) => http.get('/api/campaigns/analytics/views',
  { params, loading: models.campaigns });

export const getCampaignClickCounts = async (params) => http.get('/api/campaigns/analytics/clicks',
  { params, loading: models.campaigns });

export const getCampaignBounceCounts = async (params) => http.get('/api/campaigns/analytics/bounces',
  { params, loading: models.campaigns });

export const getCampaignLinkCounts = async (params) => http.get('/api/campaigns/analytics/links',
  { params, loading: models.campaigns });

export const convertCampaignContent = async (data) => http.post(`/api/campaigns/${data.id}/content`, data,
  { loading: models.campaigns });

export const testCampaign = async (data) => http.post(`/api/campaigns/${data.id}/test`, data,
  { loading: models.campaigns });

export const updateCampaign = async (id, data) => http.put(`/api/campaigns/${id}`, data,
  { loading: models.campaigns });

export const changeCampaignStatus = async (id, status) => http.put(`/api/campaigns/${id}/status`,
  { status }, { loading: models.campaigns });

export const updateCampaignArchive = async (id, data) => http.put(`/api/campaigns/${id}/archive`, data,
  { loading: models.campaigns });

export const deleteCampaign = async (id) => http.delete(`/api/campaigns/${id}`,
  { loading: models.campaigns });

// Media.
export const getMedia = async () => http.get('/api/media',
  { loading: models.media, store: models.media });

export const uploadMedia = (data) => http.post('/api/media', data,
  { loading: models.media });

export const deleteMedia = (id) => http.delete(`/api/media/${id}`,
  { loading: models.media });

// Templates.
export const createTemplate = async (data) => http.post('/api/templates', data,
  { loading: models.templates });

export const getTemplates = async () => http.get('/api/templates',
  { loading: models.templates, store: models.templates });

export const updateTemplate = async (data) => http.put(`/api/templates/${data.id}`, data,
  { loading: models.templates });

export const makeTemplateDefault = async (id) => http.put(`/api/templates/${id}/default`, {},
  { loading: models.templates });

export const deleteTemplate = async (id) => http.delete(`/api/templates/${id}`,
  { loading: models.templates });

// Settings.
export const getServerConfig = async () => http.get('/api/config',
  { loading: models.serverConfig, store: models.serverConfig, camelCase: false });

export const getSettings = async () => http.get('/api/settings',
  { loading: models.settings, store: models.settings, camelCase: false });

export const updateSettings = async (data) => http.put('/api/settings', data,
  { loading: models.settings });

export const testSMTP = async (data) => http.post('/api/settings/smtp/test', data,
  { loading: models.settings, disableToast: true });

export const getLogs = async () => http.get('/api/logs',
  { loading: models.logs, camelCase: false });

export const getLang = async (lang) => http.get(`/api/lang/${lang}`,
  { loading: models.lang, camelCase: false });

export const logout = async () => http.get('/api/logout', {
  auth: { username: 'wrong', password: 'wrong' },
});

export const deleteGCCampaignAnalytics = async (typ, beforeDate) => http.delete(`/api/maintenance/analytics/${typ}`,
  { loading: models.maintenance, params: { before_date: beforeDate } });

export const deleteGCSubscribers = async (typ) => http.delete(`/api/maintenance/subscribers/${typ}`,
  { loading: models.maintenance });

export const deleteGCSubscriptions = async (beforeDate) => http.delete('/api/maintenance/subscriptions/unconfirmed',
  { loading: models.maintenance, params: { before_date: beforeDate } });
