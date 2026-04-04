<script lang="ts">
  import { LayoutDashboard, Zap, Clock, Code, Database, BrainCircuit, LineChart } from 'lucide-svelte';
</script>

<div class="flex-1 flex flex-col bg-slate-900 border-r border-slate-800">
  <div class="h-14 border-b border-slate-800 flex items-center px-4 shrink-0 bg-slate-900/50 backdrop-blur-sm z-10 sticky top-0">
    <h1 class="text-sm font-semibold tracking-wide flex items-center gap-2">
      <LayoutDashboard size={16} />
      Agent Dashboard
    </h1>
  </div>
  <div class="flex-1 overflow-y-auto p-6 flex flex-col gap-6">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="bg-slate-950 border border-slate-800 rounded-xl p-4 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-indigo-500/10 text-indigo-400 flex items-center justify-center shrink-0">
          <Zap size={24} />
        </div>
        <div>
          <div class="text-2xl font-semibold text-slate-200">1,204</div>
          <div class="text-xs text-slate-500 uppercase tracking-wider">Total Actions</div>
        </div>
      </div>
      <div class="bg-slate-950 border border-slate-800 rounded-xl p-4 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-emerald-500/10 text-emerald-400 flex items-center justify-center shrink-0">
          <Clock size={24} />
        </div>
        <div>
          <div class="text-2xl font-semibold text-slate-200">14.2s</div>
          <div class="text-xs text-slate-500 uppercase tracking-wider">Avg Response Time</div>
        </div>
      </div>
      <div class="bg-slate-950 border border-slate-800 rounded-xl p-4 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-amber-500/10 text-amber-400 flex items-center justify-center shrink-0">
          <BrainCircuit size={24} />
        </div>
        <div>
          <div class="text-2xl font-semibold text-slate-200">42k</div>
          <div class="text-xs text-slate-500 uppercase tracking-wider">Tokens Processed</div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="bg-slate-950 border border-slate-800 rounded-xl p-5 flex flex-col">
        <h3 class="font-semibold text-slate-300 flex items-center gap-2 mb-4">
          <Code size={18} class="text-slate-500" />
          Top Tools Used
        </h3>
        <div class="flex-1 flex flex-col gap-3">
          <div class="flex items-center justify-between text-sm">
            <span class="text-slate-400 font-mono">bash</span>
            <div class="flex items-center gap-2 w-1/2">
              <div class="h-2 bg-slate-800 rounded-full flex-1 overflow-hidden">
                <div class="h-full bg-indigo-500 w-[80%] rounded-full"></div>
              </div>
              <span class="text-slate-300 w-8 text-right">452</span>
            </div>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-slate-400 font-mono">read_file</span>
            <div class="flex items-center gap-2 w-1/2">
              <div class="h-2 bg-slate-800 rounded-full flex-1 overflow-hidden">
                <div class="h-full bg-emerald-500 w-[65%] rounded-full"></div>
              </div>
              <span class="text-slate-300 w-8 text-right">312</span>
            </div>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-slate-400 font-mono">edit_file</span>
            <div class="flex items-center gap-2 w-1/2">
              <div class="h-2 bg-slate-800 rounded-full flex-1 overflow-hidden">
                <div class="h-full bg-amber-500 w-[40%] rounded-full"></div>
              </div>
              <span class="text-slate-300 w-8 text-right">184</span>
            </div>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-slate-400 font-mono">search_code</span>
            <div class="flex items-center gap-2 w-1/2">
              <div class="h-2 bg-slate-800 rounded-full flex-1 overflow-hidden">
                <div class="h-full bg-blue-500 w-[25%] rounded-full"></div>
              </div>
              <span class="text-slate-300 w-8 text-right">98</span>
            </div>
          </div>
        </div>
      </div>
      
      <div class="bg-slate-950 border border-slate-800 rounded-xl p-5 flex flex-col items-center justify-center text-slate-500">
        <LineChart size={48} class="mb-4 opacity-20" />
        <h3 class="font-semibold text-slate-300 mb-1">Activity Chart</h3>
        <p class="text-sm text-center opacity-70">Chart visualization component pending implementation.</p>
      </div>
    </div>
  </div>
</div>
