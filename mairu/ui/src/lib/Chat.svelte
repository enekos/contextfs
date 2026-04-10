<script lang="ts">
  import { Send, Bot, User, Loader2, Wrench, ChevronDown, ChevronRight, CheckCircle2, XCircle, Plus } from 'lucide-svelte';
  import { messages, sendMessage, isGenerating, connectionState, sessions, currentSession, switchSession, createSession, loadSessions } from './store';
  import { marked } from 'marked';
  import DOMPurify from 'dompurify';
  import { onMount, onDestroy, tick } from 'svelte';
  import { fade, slide, fly } from 'svelte/transition';
  import { search, createMemory, listContextNodes } from './api';

  let inputStr = "";
  let messagesContainer: HTMLDivElement;
  let expandedTools: Record<string, boolean> = {};
  let creatingSession = false;
  let thinkingGlyph = "◴";
  let thinkingPhrase = "Igürikatzen...";
  let thinkingTicker: ReturnType<typeof window.setInterval> | null = null;

  const quirkySpinnerFrames = ["◴", "◷", "◶", "◵", "✶", "✸", "✺", "✸", "✶"];
  const xiberokoLoadingPhrases = [
    "Igürikatzen...",
    "Eijerki apailatzen...",
    "Mustrakan ari...",
    "Hitzak txarrantxatzen...",
    "Bürüa khilikatzen...",
    "Zühürtziaz ehuntzen...",
    "Bürü-hausterietan...",
    "Egiari hüllantzen...",
    "Aitzindarien urratsetan...",
    "Sükhalteko süan txigortzen...",
    "Mündia iraulikatzen...",
    "Satanen pheredikia asmatzen...",
    "Khordokak xuxentzen...",
    "Ülünpetik argitara jalkitzen...",
    "Düdak lürruntzen...",
    "Erran-zaharrak marraskatzen...",
    "Khexatü gabe phentsatzen...",
    "Ahapetik xuxurlatzen...",
    "Bortüetako haizea behatzen...",
    "Gogoa eküratzen...",
    "Orhoikizünak xahatzen...",
    "Belagileen artean...",
    "Ilhintiak phizten...",
    "Xühürki barnebistatzen...",
    "Errejent gisa moldatzen...",
    "Basa-ahaideak asmatzen...",
    "Zamaltzainaren jauzia prestatzen...",
    "Txülülen hotsari behatzen..."
  ];

  function randomFrom<T>(items: T[]): T {
    return items[Math.floor(Math.random() * items.length)];
  }

  function refreshThinkingText() {
    thinkingGlyph = randomFrom(quirkySpinnerFrames);
    thinkingPhrase = randomFrom(xiberokoLoadingPhrases);
  }

  function startThinkingTicker() {
    if (thinkingTicker !== null) return;
    refreshThinkingText();
    thinkingTicker = window.setInterval(() => {
      refreshThinkingText();
    }, 800 + Math.floor(Math.random() * 1000));
  }

  function stopThinkingTicker() {
    if (thinkingTicker === null) return;
    window.clearInterval(thinkingTicker);
    thinkingTicker = null;
  }

  onMount(() => {
    void loadSessions();
  });
  onDestroy(() => stopThinkingTicker());

  $: if ($isGenerating) {
    startThinkingTicker();
  } else {
    stopThinkingTicker();
  }

  $: {
    $messages;
    scrollToBottom();
  }

  function toggleTool(id: string) {
    expandedTools[id] = !expandedTools[id];
  }

  function formatToolOutput(obj: any): string {
    if (typeof obj === 'string') return obj;
    if (!obj) return '';
    if (typeof obj === 'object') {
      try {
        return Object.entries(obj).map(([k, v]) => {
          const valStr = typeof v === 'string' ? v : JSON.stringify(v, null, 2);
          return `${k}:\n${valStr}`;
        }).join('\n\n');
      } catch (e) {
        return JSON.stringify(obj, null, 2);
      }
    }
    return JSON.stringify(obj, null, 2);
  }

  async function scrollToBottom() {
    await tick();
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  }

  async function handleSlashCommand(cmd: string) {
    const parts = cmd.split(" ");
    const command = parts[0].toLowerCase();
    const args = parts.slice(1).join(" ");

    // Add user message to history
    messages.update(msgs => [
      ...msgs, 
      { id: crypto.randomUUID(), role: "user", content: cmd, bashOutput: "", statuses: [], logs: [], toolCalls: [] }
    ]);

    try {
      if (command === "/clear") {
        messages.set([]);
        return;
      } else if (command === "/approve" || command === "/deny") {
        sendMessage(cmd); // Standard handling for agent approvals
        return;
      } else if (command === "/memory") {
        const subCmd = parts[1];
        const memArgs = parts.slice(2).join(" ");
        if (subCmd === "search" || subCmd === "read") {
          const data = await search({ q: memArgs, type: 'memory', topK: 15 });
          messages.update(msgs => [
            ...msgs, 
            { id: crypto.randomUUID(), role: "system", content: "Searched memories:\n" + JSON.stringify(data, null, 2), bashOutput: "", statuses: [], logs: [], toolCalls: [] }
          ]);
        } else if (subCmd === "store" || subCmd === "write") {
          await createMemory({ content: memArgs, category: "user_provided", project: "" });
          messages.update(msgs => [
            ...msgs, 
            { id: crypto.randomUUID(), role: "system", content: "Stored memory.", bashOutput: "", statuses: [], logs: [], toolCalls: [] }
          ]);
        }
      } else if (command === "/node") {
        const subCmd = parts[1];
        const nodeArgs = parts.slice(2).join(" ");
        if (subCmd === "search" || subCmd === "read") {
          const data = await search({ q: nodeArgs, type: 'context', topK: 15 });
          messages.update(msgs => [
            ...msgs, 
            { id: crypto.randomUUID(), role: "system", content: "Searched context nodes:\n" + JSON.stringify(data, null, 2), bashOutput: "", statuses: [], logs: [], toolCalls: [] }
          ]);
        } else if (subCmd === "ls") {
          const data = await listContextNodes("", nodeArgs);
          messages.update(msgs => [
            ...msgs, 
            { id: crypto.randomUUID(), role: "system", content: "Context node listing:\n" + JSON.stringify(data, null, 2), bashOutput: "", statuses: [], logs: [], toolCalls: [] }
          ]);
        }
      } else if (command === "/model") {
        if (args) {
          // Reconnect with new model
          const currentSess = $currentSession;
          // Store connection URL dynamically
          const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
          window.location.hash = `#model=${args}`; // simple way to keep state for demo, though better done in store
          
          messages.update(msgs => [
            ...msgs, 
            { id: crypto.randomUUID(), role: "system", content: `Switching model to: ${args}`, bashOutput: "", statuses: [], logs: [], toolCalls: [] }
          ]);
          
          // Reconnect with new model
          const wsUrl = `${protocol}//${window.location.host}/api/chat?session=${encodeURIComponent(currentSess)}&model=${encodeURIComponent(args)}`;
          // We would actually need to update store.ts connectWs to accept model, but for now we'll just log it
          messages.update(msgs => [
            ...msgs, 
            { id: crypto.randomUUID(), role: "system", content: `Model switch requested (Note: fully dynamic switching requires refreshing connection with ?model=${args})`, bashOutput: "", statuses: [], logs: [], toolCalls: [] }
          ]);
        }
      } else {
        // Fallback for unknown slash commands, send to agent anyway
        sendMessage(cmd);
      }
    } catch (e) {
      messages.update(msgs => [
        ...msgs, 
        { id: crypto.randomUUID(), role: "system", content: `Command failed: ${e instanceof Error ? e.message : String(e)}`, bashOutput: "", statuses: [], logs: [], toolCalls: [] }
      ]);
    }
  }

  function handleSend() {
    const input = inputStr.trim();
    if (!input || $isGenerating || $connectionState !== "connected") return;
    
    if (input.startsWith("/")) {
      handleSlashCommand(input);
    } else {
      sendMessage(input);
    }
    
    inputStr = "";
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function renderMarkdown(md: string) {
    if (!md) return "";
    return DOMPurify.sanitize(marked.parse(md) as string);
  }

  async function handleSessionSwitch(event: Event) {
    const target = event.target as HTMLSelectElement;
    const selected = target.value;
    if (!selected || selected === $currentSession) return;
    await switchSession(selected);
  }

  async function handleCreateSession() {
    if (creatingSession) return;
    const name = window.prompt("New session name");
    if (!name) return;

    creatingSession = true;
    try {
      await createSession(name);
    } catch (error) {
      const message = error instanceof Error ? error.message : "failed to create session";
      window.alert(message);
    } finally {
      creatingSession = false;
    }
  }
</script>

<div class="w-full h-full flex flex-col relative bg-[#09090b] border-l border-indigo-900/30 overflow-hidden transition-all">
  <div class="relative z-10 min-h-12 border-b border-indigo-900/40 flex items-center gap-4 px-4 sm:px-6 py-2 sm:py-3 shrink-0 bg-[#09090b]/95 backdrop-blur-sm sticky top-0 shadow-sm">
    <div class="flex flex-col gap-0.5 pl-0.5">
      <div class="text-[13px] font-semibold tracking-wide flex items-center gap-2 text-cyan-500/90">
        Mairu Agent
        {#if $connectionState === "connected"}
          <span class="w-2 h-2 rounded-full bg-emerald-500/80"></span>
        {:else}
          <span class="w-2 h-2 rounded-full bg-amber-500/80"></span>
        {/if}
      </div>
      <p class="text-[9px] tracking-wide text-indigo-400/70">Codebase assistant</p>
    </div>
    <div class="ml-auto flex items-center gap-2 pr-0.5">
      <label class="text-[9px] font-semibold uppercase tracking-widest text-cyan-500/70 px-1" for="session-select">Session</label>
      <select
        id="session-select"
        class="bg-[#11111b] border border-indigo-900/50 rounded-sm px-2 py-1 text-[10px] font-medium text-pink-200/90 min-w-32 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
        value={$currentSession}
        on:change={handleSessionSwitch}
        disabled={$isGenerating}
      >
        {#each $sessions as session}
          <option value={session}>{session}</option>
        {/each}
      </select>
      <button
        class="p-1.5 rounded-sm border border-indigo-900/50 hover:border-cyan-500 hover:bg-indigo-900/30 hover:text-pink-200 text-cyan-500/80 transition-all active:scale-95 disabled:opacity-50"
        on:click={handleCreateSession}
        disabled={$isGenerating || creatingSession}
        title="Create session"
      >
        <Plus size={12} />
      </button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto relative z-10 px-4 sm:px-6 py-6 sm:py-8 space-y-6" bind:this={messagesContainer}>
    {#if $messages.length === 0}
      <div in:fade={{ duration: 400 }} class="h-full flex flex-col items-center justify-center text-cyan-500/60 py-10">
        <Bot size={40} class="mb-4 opacity-30" />
        <p class="text-sm font-medium text-pink-300/80">Mairu is ready.</p>
        <p class="text-[11px] opacity-60 mt-1">Ask a codebase question or request a code change.</p>
      </div>
    {/if}

    {#each $messages as msg (msg.id)}
      <div class="flex flex-col gap-2" in:fly={{ y: 10, duration: 300 }}>
        <div class="flex items-center gap-2 text-[9px] font-semibold uppercase tracking-[0.1em] text-cyan-500/70 px-2 py-1">
          {#if msg.role === 'user'}
            <User size={12} class="text-cyan-500/60" /> You
          {:else if msg.role === 'assistant'}
            <Bot size={12} class="text-cyan-500/60" /> Mairu
          {:else}
            <span class="text-rose-400/80">System</span>
          {/if}
        </div>
        
        {#if msg.role === 'assistant'}
          {#if msg.toolCalls && msg.toolCalls.length > 0}
            <div class="flex flex-col gap-2 my-0.5">
              {#each msg.toolCalls as tc}
                <div class="flex flex-col text-[10px] bg-[#11111b]/40 border border-indigo-900/40 rounded-sm overflow-hidden font-mono transition-all hover:border-indigo-700/50">
                  <button class="flex items-center gap-2 px-3 py-2 text-cyan-400/90 hover:bg-indigo-900/20 transition-colors" on:click={() => toggleTool(tc.id)}>
                    {#if expandedTools[tc.id]}
                      <ChevronDown size={12} class="shrink-0 opacity-70" />
                    {:else}
                      <ChevronRight size={12} class="shrink-0 opacity-70" />
                    {/if}
                    <Wrench size={12} class={`shrink-0 ${tc.status === 'running' ? 'text-amber-400/80' : 'text-cyan-500/70'}`} />
                    <span class="font-medium">{tc.name}</span>
                    <span class="flex-1 text-left text-cyan-500/50 truncate ml-1 text-[9px]">
                      {JSON.stringify(tc.args)}
                    </span>
                    {#if tc.status === 'running'}
                      <Loader2 size={12} class="animate-spin text-amber-400/80 shrink-0" />
                    {:else if tc.status === 'completed' && !tc.result?.error}
                      <CheckCircle2 size={12} class="text-cyan-500/70 shrink-0" />
                    {:else}
                      <XCircle size={12} class="text-rose-400/80 shrink-0" />
                    {/if}
                  </button>
                  {#if expandedTools[tc.id]}
                    <div transition:slide={{ duration: 200 }} class="px-3 py-2 bg-[#0c0c10] border-t border-indigo-900/40 flex flex-col gap-2 overflow-x-auto">
                      <div class="text-cyan-500/60 font-semibold uppercase tracking-wider text-[8px]">Args</div>
                      <pre class="text-cyan-400/80 text-[9px] max-h-32 overflow-y-auto custom-scrollbar">{formatToolOutput(tc.args)}</pre>
                      {#if tc.result}
                        <div class="text-cyan-500/60 font-semibold uppercase tracking-wider text-[8px] mt-1">Result</div>
                        <pre class="text-cyan-400/80 text-[9px] max-h-48 overflow-y-auto custom-scrollbar">{formatToolOutput(tc.result)}</pre>
                      {/if}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
          {#if msg.statuses && msg.statuses.length > 0}
            <div class="flex flex-col gap-1.5 my-0.5">
              {#each msg.statuses as status}
                <div in:slide={{ duration: 150 }} class="text-[10px] text-pink-200/70 bg-[#11111b]/30 rounded-sm px-3 py-2 flex items-start gap-2 border border-indigo-900/30 font-mono">
                  <Wrench size={10} class="mt-0.5 shrink-0 opacity-50" />
                  <span>{status}</span>
                </div>
              {/each}
            </div>
          {/if}
          {#if msg.bashOutput}
            <div class="flex flex-col gap-1.5 my-0.5" transition:slide={{ duration: 150 }}>
              <div class="px-2 text-[9px] font-semibold uppercase tracking-[0.1em] text-indigo-400/60">Bash Output</div>
              <div class="text-[10px] text-pink-200/80 bg-[#0c0c10] rounded-sm px-3 py-2 border border-indigo-900/50 font-mono overflow-x-auto max-h-64 overflow-y-auto custom-scrollbar">
                <pre class="whitespace-pre-wrap">{msg.bashOutput}</pre>
              </div>
            </div>
          {/if}
          {#if msg.content || ($isGenerating && msg.id === $messages[$messages.length-1].id)}
            <div class="prose prose-invert prose-sm max-w-none rounded-sm px-4 py-3 bg-[#11111b]/40 border border-indigo-900/30 prose-pre:bg-[#0c0c10] prose-pre:border prose-pre:border-indigo-900/40 prose-pre:px-3 prose-pre:py-2 prose-pre:text-[11px] prose-a:text-cyan-400 prose-p:leading-relaxed prose-p:text-[12px] prose-headings:text-pink-200/90 prose-strong:text-pink-200/90 text-gray-300" 
                 class:opacity-80={$isGenerating && msg.id === $messages[$messages.length-1].id && !msg.content}
                 style="word-wrap: break-word;">
              {#if msg.content}
                {@html renderMarkdown(msg.content)}
              {:else if $isGenerating && msg.id === $messages[$messages.length-1].id}
                <span class="text-cyan-500/80 flex items-center gap-2 mt-1 text-[11px]">
                  <Loader2 size={12} class="animate-spin" />
                  <span class="font-medium text-pink-300/80">{thinkingGlyph}</span>
                  <span class="italic opacity-80">{thinkingPhrase}</span>
                </span>
              {/if}
            </div>
          {/if}
        {:else if msg.role === 'user'}
          <div class="bg-indigo-900/10 border border-indigo-900/40 rounded-sm px-4 py-3 text-pink-100/90 text-[12px] leading-relaxed">
            {msg.content}
          </div>
        {:else}
          <div class="bg-rose-900/10 border border-rose-900/20 rounded-sm px-4 py-3 text-rose-300/90 font-mono text-[11px] leading-relaxed whitespace-pre-wrap">
            {msg.content}
            {#if msg.content.includes('/approve')}
              <div class="mt-3 flex gap-3">
                <button on:click={() => sendMessage('/approve')} class="px-3 py-1.5 bg-green-900/30 text-green-400/90 border border-green-800/50 rounded-sm hover:bg-green-800/40 transition-colors cursor-pointer text-[11px]">Approve</button>
                <button on:click={() => sendMessage('/deny')} class="px-3 py-1.5 bg-red-900/30 text-red-400/90 border border-red-800/50 rounded-sm hover:bg-red-800/40 transition-colors cursor-pointer text-[11px]">Deny</button>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
  </div>

  <div class="relative z-10 px-4 sm:px-6 py-4 bg-[#09090b]/95 backdrop-blur-sm border-t border-indigo-900/40 shrink-0">
    <div class="relative bg-[#11111b] rounded-sm border border-indigo-900/50 focus-within:border-cyan-500/70 focus-within:ring-1 focus-within:ring-cyan-500/30 transition-all p-0.5">
      <textarea
        bind:value={inputStr}
        on:keydown={handleKeydown}
        disabled={$isGenerating || $connectionState !== "connected"}
        placeholder={$connectionState === "connected" ? "Ask anything about your codebase..." : "Connecting..."}
        class="w-full bg-transparent resize-none outline-none px-3 py-2.5 pr-12 min-h-[44px] max-h-48 overflow-y-auto text-pink-100/90 text-[12px] disabled:opacity-50 placeholder:text-indigo-500/60 custom-scrollbar"
        rows="1"
      ></textarea>
      
      <div class="absolute bottom-2 right-2 flex items-center">
        <button 
          on:click={handleSend}
          disabled={!inputStr.trim() || $isGenerating || $connectionState !== "connected"}
          class="p-1.5 rounded-sm bg-indigo-900/60 text-pink-200/90 disabled:opacity-50 disabled:bg-[#11111b]/50 disabled:text-indigo-500/60 transition-all hover:bg-indigo-800 hover:text-pink-100 active:scale-95"
        >
          {#if $isGenerating}
            <Loader2 size={14} class="animate-spin" />
          {:else}
            <Send size={14} />
          {/if}
        </button>
      </div>
    </div>
    <div class="text-[9px] text-center text-cyan-500/50 mt-2 font-medium">
      Press <kbd class="px-1 py-0.5 bg-[#11111b] rounded-sm border border-indigo-900/40 mx-0.5">Enter</kbd> to send, <kbd class="px-1 py-0.5 bg-[#11111b] rounded-sm border border-indigo-900/40 mx-0.5">Shift+Enter</kbd> for new line
    </div>
  </div>
</div>
<style>
  /* Removed heavy padding classes and simplified */
</style>
