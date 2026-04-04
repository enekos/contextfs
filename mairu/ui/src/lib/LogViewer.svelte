<script lang="ts">
  import { Terminal, Database, Cpu, Activity, ShieldAlert, FileCode } from 'lucide-svelte';
  import { connectionState, messages } from './store';

  type TabType = "overview" | "logs" | "graph";
  let activeTab: TabType = "overview";
</script>

<div class="flex-1 flex flex-col bg-slate-950">
  <div class="h-14 border-b border-slate-800 flex items-center px-4 shrink-0 bg-slate-950 text-slate-400">
    <div class="flex gap-4 h-full">
      <button 
        class="flex items-center gap-2 hover:text-slate-200 transition-colors h-14 {activeTab === 'overview' ? 'text-indigo-400 border-b-2 border-indigo-400' : 'text-slate-500 border-b-2 border-transparent'}"
        on:click={() => activeTab = 'overview'}
      >
        <Activity size={16} /> Overview
      </button>
      <button 
        class="flex items-center gap-2 hover:text-slate-200 transition-colors h-14 {activeTab === 'logs' ? 'text-indigo-400 border-b-2 border-indigo-400' : 'text-slate-500 border-b-2 border-transparent'}"
        on:click={() => activeTab = 'logs'}
      >
        <Terminal size={16} /> Logs
      </button>
      <button 
        class="flex items-center gap-2 hover:text-slate-200 transition-colors h-14 {activeTab === 'graph' ? 'text-indigo-400 border-b-2 border-indigo-400' : 'text-slate-500 border-b-2 border-transparent'}"
        on:click={() => activeTab = 'graph'}
      >
        <Database size={16} /> Graph
      </button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto p-8">
    {#if activeTab === 'overview'}
      <div class="max-w-3xl mx-auto space-y-8">
        <div>
          <h2 class="text-2xl font-semibold mb-2">Workspace Overview</h2>
          <p class="text-slate-400">Mairu is connected and ready to navigate your codebase.</p>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div class="p-6 rounded-2xl bg-slate-900 border border-slate-800">
            <div class="w-10 h-10 rounded-lg bg-emerald-500/10 text-emerald-500 flex items-center justify-center mb-4">
              <Cpu size={20} />
            </div>
            <h3 class="font-semibold mb-1">Mairu Agent</h3>
            <div class="text-sm text-slate-400 flex items-center gap-2">
              Status: 
              {#if $connectionState === "connected"}
                <span class="text-emerald-400">Online</span>
              {:else}
                <span class="text-amber-400">Connecting...</span>
              {/if}
            </div>
          </div>

          <div class="p-6 rounded-2xl bg-slate-900 border border-slate-800">
            <div class="w-10 h-10 rounded-lg bg-blue-500/10 text-blue-500 flex items-center justify-center mb-4">
              <Database size={20} />
            </div>
            <h3 class="font-semibold mb-1">Code Graph</h3>
            <div class="text-sm text-slate-400">
              Backed by Meilisearch
            </div>
          </div>
        </div>

        <div class="p-6 rounded-2xl bg-indigo-500/5 border border-indigo-500/20">
          <h3 class="font-semibold text-indigo-400 mb-4 flex items-center gap-2">
            <ShieldAlert size={18} /> Capabilities
          </h3>
          <ul class="space-y-3 text-sm text-slate-300">
            <li class="flex items-start gap-3">
              <div class="w-1.5 h-1.5 rounded-full bg-indigo-500 mt-1.5"></div>
              <div>
                <strong class="text-slate-200">Surgical Reading</strong> <br/>
                Reads specific AST nodes instead of dumping entire files into context.
              </div>
            </li>
            <li class="flex items-start gap-3">
              <div class="w-1.5 h-1.5 rounded-full bg-indigo-500 mt-1.5"></div>
              <div>
                <strong class="text-slate-200">Multi-Agent Dispatch</strong> <br/>
                Spawns sub-agents to parallelize codebase research.
              </div>
            </li>
            <li class="flex items-start gap-3">
              <div class="w-1.5 h-1.5 rounded-full bg-indigo-500 mt-1.5"></div>
              <div>
                <strong class="text-slate-200">Terminal Native</strong> <br/>
                Executes bash commands, tests, and git operations autonomously.
              </div>
            </li>
          </ul>
        </div>
      </div>
    {:else if activeTab === 'logs'}
      <div class="max-w-3xl mx-auto h-full flex flex-col font-mono text-sm">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-xl font-semibold font-sans">System Logs</h2>
        </div>
        <div class="flex-1 bg-slate-900 border border-slate-800 rounded-xl p-4 overflow-y-auto space-y-2">
          {#each $messages as msg}
            {#each msg.statuses as status}
              <div class="text-slate-400 border-l-2 border-indigo-500 pl-3 py-1 bg-slate-950/50 rounded-r">
                <span class="text-indigo-400">[{new Date().toLocaleTimeString()}]</span> {status}
              </div>
            {/each}
            {#each msg.logs as log}
              <div class="text-slate-400 border-l-2 border-amber-500 pl-3 py-1 bg-slate-950/50 rounded-r">
                <span class="text-amber-400">[{new Date().toLocaleTimeString()}]</span> {log}
              </div>
            {/each}
            {#each msg.toolCalls as tc}
              <div class="text-slate-400 border-l-2 border-emerald-500 pl-3 py-1 bg-slate-950/50 rounded-r">
                <span class="text-emerald-400">[{new Date().toLocaleTimeString()}]</span> 
                TOOL CALL: {tc.name} 
                <span class="text-slate-500 text-xs">({tc.status})</span>
                {#if tc.status === 'error'}
                  <div class="text-rose-400 text-xs mt-1 bg-rose-500/10 p-2 rounded">
                    {JSON.stringify(tc.result)}
                  </div>
                {/if}
              </div>
            {/each}
          {/each}
          {#if $messages.length === 0 || ($messages.every(m => m.statuses.length === 0 && m.toolCalls.length === 0 && m.logs.length === 0))}
            <div class="text-slate-500 text-center py-8 font-sans">No recent activity logs available.</div>
          {/if}
        </div>
      </div>
    {:else if activeTab === 'graph'}
      <div class="max-w-3xl mx-auto h-full flex flex-col">
        <h2 class="text-xl font-semibold mb-4">Code Graph Explorer</h2>
        <div class="flex-1 bg-slate-900 border border-slate-800 rounded-xl flex items-center justify-center text-slate-500 p-8">
          <div class="text-center">
            <Database size={48} class="mx-auto mb-4 opacity-20" />
            <h3 class="text-lg font-semibold text-slate-300 mb-2">Graph Visualization Not Connected</h3>
            <p class="max-w-md mx-auto text-sm">Connect to a live Meilisearch instance to see context nodes and vector representations of your codebase.</p>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
