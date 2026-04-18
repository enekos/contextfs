<script lang="ts">
  import { onMount } from "svelte";
  import { connectChat, activeView } from "./lib/store";
  import Sidebar from "./lib/Sidebar.svelte";
  import Chat from "./lib/Chat.svelte";
  import Workspace from "./lib/Workspace.svelte";
  import Dashboard from "./lib/Dashboard.svelte";
  import LogViewer from "./lib/LogViewer.svelte";
  import Settings from "./lib/Settings.svelte";
  import "./app.css";

  onMount(() => {
    connectChat();

    // Check url hash for view/model states on load
    const hash = window.location.hash;
    if (hash) {
      if (hash.includes('view=')) {
        const view = hash.split('view=')[1].split('&')[0];
        if (view === 'chat' || view === 'workspace' || view === 'dashboard' || view === 'logs' || view === 'settings') {
          activeView.set(view);
        }
      }
    }
  });
</script>

<div class="flex h-screen bg-black text-green-500 overflow-hidden font-sans">
  <Sidebar />
  <div class="flex-1 flex flex-col relative overflow-hidden">
    {#if $activeView === 'chat'}
      <Chat />
    {:else if $activeView === 'workspace'}
      <Workspace />
    {:else if $activeView === 'dashboard'}
      <Dashboard />
    {:else if $activeView === 'logs'}
      <LogViewer />
    {:else if $activeView === 'settings'}
      <Settings />
    {/if}
  </div>
</div>
