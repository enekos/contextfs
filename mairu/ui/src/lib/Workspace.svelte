<script lang="ts">
  import { FolderGit2, Search, FileCode, FolderOpen, MoreVertical, RefreshCw } from 'lucide-svelte';
</script>

<div class="flex-1 flex flex-col bg-green-950 border-r border-green-900">
  <div class="h-14 border-b border-green-900 flex items-center px-4 justify-between shrink-0 bg-green-950/50 backdrop-blur-sm z-10 sticky top-0">
    <h1 class="text-sm font-semibold tracking-wide flex items-center gap-2">
      <FolderGit2 size={16} />
      Workspace Context
    </h1>
    <div class="flex items-center gap-2">
      <div class="relative">
        <Search size={14} class="absolute left-2.5 top-2 text-green-600" />
        <input 
          type="text" 
          placeholder="Search files..." 
          class="bg-black border border-green-900 rounded-md pl-8 pr-3 py-1 text-xs text-green-300 focus:outline-none focus:border-green-500 w-48 transition-colors"
        />
      </div>
      <button class="p-1.5 text-green-600 hover:text-green-400 hover:bg-green-900 rounded-md transition-colors">
        <RefreshCw size={14} />
      </button>
    </div>
  </div>
  
  <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4 text-sm font-mono">
    <div class="text-xs font-semibold text-green-600 uppercase tracking-wider mb-2 font-sans">Project Explorer</div>
    
    <div class="flex flex-col gap-1">
      <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-400">
        <FolderOpen size={16} class="text-green-400" />
        <span>mairu/</span>
      </div>
      <div class="flex flex-col gap-1 pl-4 border-l border-green-900 ml-2">
        <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-400">
          <FolderOpen size={16} class="text-green-400" />
          <span>internal/</span>
        </div>
        <div class="flex flex-col gap-1 pl-4 border-l border-green-900 ml-2">
          <div class="flex items-center justify-between px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-500 group">
            <div class="flex items-center gap-2">
              <FolderOpen size={16} class="text-green-400" />
              <span>agent/</span>
            </div>
          </div>
          <div class="flex items-center justify-between px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-500 group">
            <div class="flex items-center gap-2">
              <FolderOpen size={16} class="text-green-400" />
              <span>web/</span>
            </div>
          </div>
        </div>
        
        <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-400">
          <FolderOpen size={16} class="text-green-400" />
          <span>ui/</span>
        </div>
        <div class="flex flex-col gap-1 pl-4 border-l border-green-900 ml-2">
          <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-400">
            <FolderOpen size={16} class="text-amber-400" />
            <span>src/</span>
          </div>
          <div class="flex flex-col gap-1 pl-4 border-l border-green-900 ml-2">
            <div class="flex items-center gap-2 px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-400">
              <FolderOpen size={16} class="text-amber-400" />
              <span>lib/</span>
            </div>
            <div class="flex flex-col gap-1 pl-4 border-l border-green-900 ml-2">
              <div class="flex items-center justify-between px-2 py-1.5 bg-indigo-500/10 border border-green-500/20 rounded-md cursor-pointer text-indigo-300 group">
                <div class="flex items-center gap-2">
                  <FileCode size={16} class="text-orange-400" />
                  <span>Workspace.svelte</span>
                </div>
                <span class="text-[10px] bg-indigo-500/20 px-1.5 py-0.5 rounded text-green-400 font-sans">Active</span>
              </div>
              <div class="flex items-center justify-between px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-500 group">
                <div class="flex items-center gap-2">
                  <FileCode size={16} class="text-orange-400" />
                  <span>Chat.svelte</span>
                </div>
              </div>
              <div class="flex items-center justify-between px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-500 group">
                <div class="flex items-center gap-2">
                  <FileCode size={16} class="text-blue-400" />
                  <span>store.ts</span>
                </div>
              </div>
            </div>
            <div class="flex items-center px-2 py-1.5 hover:bg-green-900/50 rounded-md cursor-pointer text-green-500">
              <FileCode size={16} class="text-orange-400" />
              <span>App.svelte</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
