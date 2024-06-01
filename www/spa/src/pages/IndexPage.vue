<template>
  <q-page class="flex flex-center">
    <stream-table />
  </q-page>
</template>

<script setup>
import StreamTable from 'components/StreamTable.vue';

defineOptions({
  name: 'IndexPage',
});
</script>
