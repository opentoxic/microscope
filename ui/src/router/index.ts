import { createRouter, createWebHistory } from 'vue-router'
import EntryListView from '../views/EntryListView.vue'
import EntryDetailView from '../views/EntryDetailView.vue'
import SettingsView from '../views/SettingsView.vue'

const router = createRouter({
  history: createWebHistory('/microscope/'),
  routes: [
    { path: '/', name: 'list', component: EntryListView },
    { path: '/entries/:id', name: 'detail', component: EntryDetailView, props: true },
    { path: '/settings', name: 'settings', component: SettingsView },
  ],
})

export default router
