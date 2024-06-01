<template>
  <q-table :rows="streams" :columns="columns" row-key="name">
    <template v-slot:body-cell-commands="props">
      <q-td :props="props">
        <q-btn label="Delete" @click="deleteStream(props.row.name)" />
      </q-td>
    </template>
  </q-table>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue';
import { useStreamStore } from 'src/stores/streamStore';

const streamStore = useStreamStore();
const streams = ref([]);

onMounted(async () => {
  await streamStore.fetchStreams();
  streams.value = streamStore.streams;
});

watch(() => streamStore.streams, (newStreams) => {
  streams.value = newStreams;
});

const columns = [
  {
    name: 'name', label: 'Name', align: 'left', field: 'name',
  },
  {
    name: 'online', label: 'Online', align: 'center', field: 'online',
  },
  {
    name: 'commands', label: 'Commands', align: 'center', field: 'commands',
  },
];

const deleteStream = async (name) => {
  try {
    await streamStore.removeStream(name);
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Error deleting stream: %s', error.message);
  }
};
</script>
