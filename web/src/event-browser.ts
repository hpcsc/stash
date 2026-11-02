import { mount } from 'svelte'
import './app.css'
import EventBrowser from './lib/components/EventBrowser.svelte'

const app = mount(EventBrowser, {
  target: document.getElementById('app')!,
})

export default app
