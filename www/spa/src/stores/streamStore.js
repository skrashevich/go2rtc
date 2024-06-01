/* eslint-disable no-console */
// src/stores/streamStore.js
import { defineStore } from 'pinia';
import { getStreams, addStream, deleteStream } from 'src/services/api';

export const useStreamStore = defineStore('stream', {
  state: () => ({
    streams: [],
  }),
  actions: {
    async fetchStreams() {
      try {
        const response = await getStreams();
        this.streams = response.data;
      } catch (error) {
        console.error('Error fetching streams:', error.message || error);
      }
    },
    async createStream(stream) {
      if (!stream || typeof stream !== 'object') {
        console.error('Invalid stream object');
        return;
      }

      try {
        await addStream(stream);
        // Optimistically update the state instead of refetching
        this.streams.push(stream);
      } catch (error) {
        console.error('Error adding stream:', error.message || error);
      }
    },
    async removeStream(name) {
      if (!name || typeof name !== 'string') {
        console.error('Invalid stream name');
        return;
      }

      try {
        await deleteStream(name);
        // Optimistically update the state instead of refetching
        this.streams = this.streams.filter((stream) => stream.name !== name);
      } catch (error) {
        console.error('Error deleting stream:', error.message || error);
      }
    },
  },
});
