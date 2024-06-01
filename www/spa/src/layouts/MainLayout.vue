<template>
  <q-layout>
    <q-header elevated>
      <q-toolbar>
        <q-toolbar-title>Stream Manager</q-toolbar-title>
        <q-btn flat round icon="menu" @click="toggleLeftDrawer" />
      </q-toolbar>
    </q-header>

    <q-drawer show-if-above v-model="leftDrawerOpen" side="left" width="250px">
      <q-list>
        <q-item to="/" clickable v-ripple>
          <q-item-section>Home</q-item-section>
        </q-item>
        <q-item to="/add" clickable v-ripple>
          <q-item-section>Add Stream</q-item-section>
        </q-item>
        <q-item to="/config" clickable v-ripple>
          <q-item-section>Configuration</q-item-section>
        </q-item>
        <q-item to="/log" clickable v-ripple>
          <q-item-section>Logs</q-item-section>
        </q-item>
      </q-list>
    </q-drawer>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { ref } from 'vue';

const leftDrawerOpen = ref(false);

const toggleLeftDrawer = () => {
  leftDrawerOpen.value = !leftDrawerOpen.value;
};
</script>
