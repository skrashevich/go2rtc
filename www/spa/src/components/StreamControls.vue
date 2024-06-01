<template>
    <div class="controls">
      <q-btn label="Stream" @click="stream" />
      <q-toggle v-model="darkMode" label="Dark Mode" @update:model-value="toggleDarkMode" />
    </div>
  </template>

<script setup>
import { ref } from 'vue';
import { useDarkModeStore } from 'src/stores/darkModeStore';

const darkModeStore = useDarkModeStore();
const darkMode = ref(darkModeStore.darkMode);

const toggleDarkMode = () => {
  darkModeStore.toggleDarkMode();
};
</script>
