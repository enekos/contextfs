<script lang="ts">
  import { Settings as SettingsIcon, Save, Key, Cpu, ShieldAlert, Bot } from 'lucide-svelte';
</script>

<div class="flex-1 flex flex-col bg-slate-900 border-r border-slate-800">
  <div class="h-14 border-b border-slate-800 flex items-center px-4 justify-between shrink-0 bg-slate-900/50 backdrop-blur-sm z-10 sticky top-0">
    <h1 class="text-sm font-semibold tracking-wide flex items-center gap-2">
      <SettingsIcon size={16} />
      Agent Settings
    </h1>
    <button class="flex items-center gap-2 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-md text-sm font-medium transition-colors shadow-sm">
      <Save size={14} />
      Save Changes
    </button>
  </div>
  
  <div class="flex-1 overflow-y-auto p-6">
    <div class="max-w-2xl mx-auto space-y-8">
      
      <!-- LLM Configuration -->
      <section>
        <h2 class="text-lg font-semibold text-slate-200 mb-4 flex items-center gap-2 border-b border-slate-800 pb-2">
          <Cpu size={18} class="text-indigo-400" />
          Model Configuration
        </h2>
        
        <div class="space-y-4">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <label class="text-sm font-medium text-slate-400">Primary Provider</label>
              <select class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors">
                <option value="gemini">Google Gemini</option>
                <option value="anthropic">Anthropic Claude</option>
                <option value="openai">OpenAI</option>
                <option value="local">Local (Ollama/LM Studio)</option>
              </select>
            </div>
            <div class="space-y-1.5">
              <label class="text-sm font-medium text-slate-400">Model Name</label>
              <select class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors">
                <option value="gemini-2.5-pro">Gemini 2.5 Pro</option>
                <option value="gemini-2.5-flash">Gemini 2.5 Flash</option>
              </select>
            </div>
          </div>
          
          <div class="space-y-1.5">
            <label class="text-sm font-medium text-slate-400 flex items-center gap-2">
              <Key size={14} /> API Key
            </label>
            <div class="flex gap-2">
              <input type="password" value="************************" class="flex-1 bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors" />
            </div>
            <p class="text-xs text-slate-500 mt-1">Leave blank to use environment variables (e.g. GEMINI_API_KEY).</p>
          </div>
        </div>
      </section>

      <!-- System Prompt -->
      <section>
        <h2 class="text-lg font-semibold text-slate-200 mb-4 flex items-center gap-2 border-b border-slate-800 pb-2">
          <Bot size={18} class="text-emerald-400" />
          Agent Identity
        </h2>
        
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-slate-400">Custom System Instructions</label>
          <textarea 
            rows="4" 
            class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors resize-none font-mono text-xs"
            placeholder="Additional instructions for the agent..."
          >You are Mairu, an expert software engineering assistant.
You specialize in navigating the contextfs workspace.</textarea>
          <p class="text-xs text-slate-500 mt-1">These instructions are appended to the base system prompt.</p>
        </div>
      </section>

      <!-- Security Settings -->
      <section>
        <h2 class="text-lg font-semibold text-slate-200 mb-4 flex items-center gap-2 border-b border-slate-800 pb-2">
          <ShieldAlert size={18} class="text-amber-400" />
          Security & Permissions
        </h2>
        
        <div class="space-y-3 bg-slate-950 border border-slate-800 rounded-xl p-4">
          <label class="flex items-start gap-3 cursor-pointer">
            <div class="relative flex items-center mt-0.5">
              <input type="checkbox" checked class="peer sr-only" />
              <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500"></div>
            </div>
            <div>
              <div class="text-sm font-medium text-slate-200">Require approval for bash commands</div>
              <div class="text-xs text-slate-500">Agent will pause and ask before executing any terminal command.</div>
            </div>
          </label>
          
          <label class="flex items-start gap-3 cursor-pointer">
            <div class="relative flex items-center mt-0.5">
              <input type="checkbox" checked class="peer sr-only" />
              <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500"></div>
            </div>
            <div>
              <div class="text-sm font-medium text-slate-200">Require approval for file modifications</div>
              <div class="text-xs text-slate-500">Agent will pause and ask before writing or editing files.</div>
            </div>
          </label>
        </div>
      </section>

    </div>
  </div>
</div>
