import { mount } from 'svelte'
import './app.css'
import EventBrowserApp from './lib/components/EventBrowserApp.svelte'

const app = mount(EventBrowserApp, {
  target: document.getElementById('app')!,
})

export default app
