<script lang="ts">
  import { Code2, MessageSquare, Settings, LayoutDashboard } from 'lucide-svelte';
  import { activeView, type ViewType } from './store';

  function setView(view: ViewType) {
    activeView.set(view);
  }
</script>

<div class="w-14 flex flex-col items-center py-4 bg-slate-950 border-r border-slate-800 shrink-0">
  <div class="mb-8">
    <div class="w-8 h-8 rounded-lg bg-indigo-500 flex items-center justify-center text-white font-bold shadow-lg shadow-indigo-500/20">
      M
    </div>
  </div>

  <nav class="flex-1 flex flex-col gap-4 w-full items-center">
    <button 
      class="p-3 rounded-xl transition-colors {$activeView === 'chat' ? 'text-indigo-400 bg-slate-900' : 'text-slate-500 hover:text-slate-300'}"
      on:click={() => setView('chat')}
      title="Chat"
    >
      <MessageSquare size={20} />
    </button>
    <button 
      class="p-3 rounded-xl transition-colors {$activeView === 'workspace' ? 'text-indigo-400 bg-slate-900' : 'text-slate-500 hover:text-slate-300'}"
      on:click={() => setView('workspace')}
      title="Workspace Context"
    >
      <Code2 size={20} />
    </button>
    <button 
      class="p-3 rounded-xl transition-colors {$activeView === 'dashboard' ? 'text-indigo-400 bg-slate-900' : 'text-slate-500 hover:text-slate-300'}"
      on:click={() => setView('dashboard')}
      title="Dashboard"
    >
      <LayoutDashboard size={20} />
    </button>
  </nav>

  <div class="mt-auto">
    <button 
      class="p-3 rounded-xl transition-colors {$activeView === 'settings' ? 'text-indigo-400 bg-slate-900' : 'text-slate-500 hover:text-slate-300'}"
      on:click={() => setView('settings')}
      title="Settings"
    >
      <Settings size={20} />
    </button>
  </div>
</div>
