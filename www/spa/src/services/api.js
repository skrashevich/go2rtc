// src/services/api.js
import { api } from 'boot/axios';

export const getStreams = () => api.get('/streams');
export const addStream = (data) => api.put('/streams', data);
export const deleteStream = (name) => api.delete(`/streams?src=${name}`);
export const getConfig = () => api.get('/config');
export const saveConfig = (data) => api.post('/config', data);
export const restartService = () => api.post('/restart');
export const getLogs = () => api.get('/log');
export const cleanLogs = () => api.delete('/log');
export const toggleDarkMode = () => api.post('/dark-mode/toggle');
