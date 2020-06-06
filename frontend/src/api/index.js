import { ToastProgrammatic as Toast } from 'buefy';
import axios from 'axios';
import humps from 'humps';
import qs from 'qs';
import store from '../store';
import { models } from '../constants';

const http = axios.create({
  baseURL: process.env.BASE_URL,
  withCredentials: false,
  responseType: 'json',
  transformResponse: [
    // Apply the defaut transformations as well.
    ...axios.defaults.transformResponse,
    (resp) => {
      if (!resp) {
        return resp;
      }

      // There's an error message.
      if ('message' in resp && resp.message !== '') {
        return resp;
      }

      const data = humps.camelizeKeys(resp.data);
      return data;
    },
  ],

  // Override the default serializer to switch params from becoming []id=a&[]id=b ...
  // in GET and DELETE requests to id=a&id=b.
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});


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

  // Store the API response for a model.
  if ('store' in resp.config) {
    store.commit('setModelResponse', { model: resp.config.store, data: resp.data });
  }
  return resp;
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

  Toast.open({
    message: msg,
    type: 'is-danger',
    queue: false,
  });

  return Promise.reject(err);
});

// API calls accept the following config keys.
// loading: modelName (set's the loading status in the global store: eg: store.loading.lists = true)
// store: modelName (set's the API response in the global store. eg: store.lists: { ... } )

// Lists.
export const getLists = () => http.get('/api/lists',
  { loading: models.lists, store: models.lists });

export const createList = (data) => http.post('/api/lists', data,
  { loading: models.lists });

export const updateList = (data) => http.put(`/api/lists/${data.id}`, data,
  { loading: models.lists });

export const deleteList = (id) => http.delete(`/api/lists/${id}`,
  { loading: models.lists });

// Subscribers.
export const getSubscribers = async (params) => http.get('/api/subscribers',
  { params, loading: models.subscribers, store: models.subscribers });

export const createSubscriber = (data) => http.post('/api/subscribers', data,
  { loading: models.subscribers });

export const updateSubscriber = (data) => http.put(`/api/subscribers/${data.id}`, data,
  { loading: models.subscribers });

export const deleteSubscriber = (id) => http.delete(`/api/subscribers/${id}`,
  { loading: models.subscribers });

export const addSubscribersToLists = (data) => http.put('/api/subscribers/lists', data,
  { loading: models.subscribers });

export const addSubscribersToListsByQuery = (data) => http.put('/api/subscribers/query/lists',
  data, { loading: models.subscribers });

export const blacklistSubscribers = (data) => http.put('/api/subscribers/blacklist', data,
  { loading: models.subscribers });

export const blacklistSubscribersByQuery = (data) => http.put('/api/subscribers/query/blacklist', data,
  { loading: models.subscribers });

export const deleteSubscribers = (params) => http.delete('/api/subscribers',
  { params, loading: models.subscribers });

export const deleteSubscribersByQuery = (data) => http.post('/api/subscribers/query/delete', data,
  { loading: models.subscribers });

// Subscriber import.
export const importSubscribers = (data) => http.post('/api/import/subscribers', data);

export const getImportStatus = () => http.get('/api/import/subscribers');

export const getImportLogs = () => http.get('/api/import/subscribers/logs');

export const stopImport = () => http.delete('/api/import/subscribers');

// Campaigns.
export const getCampaigns = async (params) => http.get('/api/campaigns',
  { params, loading: models.campaigns, store: models.campaigns });

export const getCampaign = async (id) => http.get(`/api/campaigns/${id}`,
  { loading: models.campaigns });

export const getCampaignStats = async () => http.get('/api/campaigns/running/stats', {});

export const createCampaign = async (data) => http.post('/api/campaigns', data,
  { loading: models.campaigns });

export const testCampaign = async (data) => http.post(`/api/campaigns/${data.id}/test`, data,
  { loading: models.campaigns });

export const updateCampaign = async (id, data) => http.put(`/api/campaigns/${id}`, data,
  { loading: models.campaigns });

export const changeCampaignStatus = async (id, status) => http.put(`/api/campaigns/${id}/status`,
  { status }, { loading: models.campaigns });

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
