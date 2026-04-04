<script lang="ts">
  import { FolderGit2, Search, FileCode, FolderOpen, MoreVertical, RefreshCw } from 'lucide-svelte';
</script>

<div class="flex-1 flex flex-col bg-slate-900 border-r border-slate-800">
  <div class="h-14 border-b border-slate-800 flex items-center px-4 justify-between shrink-0 bg-slate-900/50 backdrop-blur-sm z-10 sticky top-0">
    <h1 class="text-sm font-semibold tracking-wide flex items-center gap-2">
      <FolderGit2 size={16} />
      Workspace Context
    </h1>
    <div class="flex items-center gap-2">
      <div class="relative">
        <Search size={14} class="absolute left-2.5 top-2 text-slate-500" />
        <input 
          type="text" 
          placeholder="Search files..." 
          class="bg-slate-950 border border-slate-800 rounded-md pl-8 pr-3 py-1 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 w-48 transition-colors"
        />
      </div>
      <button class="p-1.5 text-slate-500 hover:text-slate-300 hover:bg-slate-800 rounded-md transition-colors">
        <RefreshCw size={14} />
      </button>
    </div>
  </div>
  
  <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4 text-sm font-mono">
    <div class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 font-sans">Project Explorer</div>
    
    <div class="flex flex-col gap-1">
      <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-300">
        <FolderOpen size={16} class="text-indigo-400" />
        <span>mairu/</span>
      </div>
      <div class="flex flex-col gap-1 pl-4 border-l border-slate-800 ml-2">
        <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-300">
          <FolderOpen size={16} class="text-indigo-400" />
          <span>internal/</span>
        </div>
        <div class="flex flex-col gap-1 pl-4 border-l border-slate-800 ml-2">
          <div class="flex items-center justify-between px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-400 group">
            <div class="flex items-center gap-2">
              <FolderOpen size={16} class="text-indigo-400" />
              <span>agent/</span>
            </div>
          </div>
          <div class="flex items-center justify-between px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-400 group">
            <div class="flex items-center gap-2">
              <FolderOpen size={16} class="text-indigo-400" />
              <span>web/</span>
            </div>
          </div>
        </div>
        
        <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-300">
          <FolderOpen size={16} class="text-indigo-400" />
          <span>ui/</span>
        </div>
        <div class="flex flex-col gap-1 pl-4 border-l border-slate-800 ml-2">
          <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-300">
            <FolderOpen size={16} class="text-amber-400" />
            <span>src/</span>
          </div>
          <div class="flex flex-col gap-1 pl-4 border-l border-slate-800 ml-2">
            <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-300">
              <FolderOpen size={16} class="text-amber-400" />
              <span>lib/</span>
            </div>
            <div class="flex flex-col gap-1 pl-4 border-l border-slate-800 ml-2">
              <div class="flex items-center justify-between px-2 py-1.5 bg-indigo-500/10 border border-indigo-500/20 rounded-md cursor-pointer text-indigo-300 group">
                <div class="flex items-center gap-2">
                  <FileCode size={16} class="text-orange-400" />
                  <span>Workspace.svelte</span>
                </div>
                <span class="text-[10px] bg-indigo-500/20 px-1.5 py-0.5 rounded text-indigo-400 font-sans">Active</span>
              </div>
              <div class="flex items-center justify-between px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-400 group">
                <div class="flex items-center gap-2">
                  <FileCode size={16} class="text-orange-400" />
                  <span>Chat.svelte</span>
                </div>
              </div>
              <div class="flex items-center justify-between px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-400 group">
                <div class="flex items-center gap-2">
                  <FileCode size={16} class="text-blue-400" />
                  <span>store.ts</span>
                </div>
              </div>
            </div>
            <div class="flex items-center px-2 py-1.5 hover:bg-slate-800/50 rounded-md cursor-pointer text-slate-400">
              <FileCode size={16} class="text-orange-400" />
              <span>App.svelte</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
