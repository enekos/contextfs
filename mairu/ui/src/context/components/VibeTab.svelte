<script lang="ts">
  import { fmtDate, categoryColors, scoreColor, impColor } from "../lib/utils";
  import { vibeMutationPlan, vibeMutationExecute } from "../../lib/api";

  let prompt = "";
  let project = "";
  let topK = 5;
  let loading = false;
  let error = "";

  // Mutation state
  let mutationPlan: { reasoning: string; operations: Array<{ op: string; target?: string; description: string; data: Record<string, any> }> } | null = null;
  let selectedOps: boolean[] = [];
  let executing = false;
  let executionResults: Array<{ op: string; result?: string; error?: string }> | null = null;

  // History
  let history: Array<{ prompt: string; timestamp: Date; reasoning: string }> = [];

  async function runVibeMutation() {
    if (!prompt.trim()) return;
    loading = true; error = "";
    try {
      const data = await vibeMutationPlan(prompt, project || "", topK);
      mutationPlan = data;
      selectedOps = (mutationPlan?.operations || []).map(() => true);
      history = [{ prompt, timestamp: new Date(), reasoning: mutationPlan?.reasoning || "" }, ...history.slice(0, 19)];
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function executeApproved() {
    if (!mutationPlan) return;
    const approved = mutationPlan.operations.filter((_, i) => selectedOps[i]);
    if (approved.length === 0) return;
    executing = true; error = "";
    try {
      const data = await vibeMutationExecute(approved, project || "");
      executionResults = data.results;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      executing = false;
    }
  }

  function keydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); runVibeMutation(); }
  }

  function toggleAll() {
    const allSelected = selectedOps.every(Boolean);
    selectedOps = selectedOps.map(() => !allSelected);
  }

  function opColor(op: string): string {
    if (op.startsWith("create")) return "#22c55e";
    if (op.startsWith("delete")) return "#ef4444";
    return "#f59e0b";
  }

  function opSymbol(op: string): string {
    if (op.startsWith("create")) return "+";
    if (op.startsWith("delete")) return "-";
    return "~";
  }

  $: approvedCount = selectedOps.filter(Boolean).length;
</script>

<section class="vibe-section">
  <!-- Header -->
  <div class="vibe-header">
    <h2 class="vibe-title">Vibe Mutation</h2>
    <p class="vibe-desc">
      Describe changes in plain English. The LLM plans mutations, you review and approve.
    </p>
  </div>

  <div class="vibe-controls">
    <div class="vibe-input-row">
      <textarea
        class="vibe-input"
        placeholder="What do you want to change? e.g. 'Mark all testing memories as importance 8'"
        bind:value={prompt}
        on:keydown={keydown}
        disabled={loading || executing}
        rows="2"
      ></textarea>

      <div class="vibe-input-actions">
        <div class="vibe-opts">
          <label>
            Project
            <input type="text" class="vibe-opt-input" placeholder="(all)" bind:value={project} />
          </label>
          <label>
            Top K
            <input type="number" class="vibe-opt-input vibe-opt-num" min="1" max="50" bind:value={topK} />
          </label>
        </div>
        <button
          class="btn-primary vibe-submit"
          on:click={runVibeMutation}
          disabled={loading || executing || !prompt.trim()}
        >
          {#if loading}
            Thinking...
          {:else}
            Plan
          {/if}
        </button>
      </div>
    </div>
  </div>

  {#if error}
    <div class="vibe-error">
      <strong>Error:</strong> {error}
      <button on:click={() => error = ""}>x</button>
    </div>
  {/if}

  <!-- Mutation Plan -->
  {#if mutationPlan}
    <div class="vibe-reasoning">
      <span class="vibe-reasoning-label">Plan</span>
      {mutationPlan.reasoning}
    </div>

    {#if mutationPlan.operations.length === 0}
      <p class="vibe-no-results">No mutations needed. The LLM determined no changes are necessary.</p>
    {:else}
      <div class="vibe-mutation-header">
        <span>{mutationPlan.operations.length} operation{mutationPlan.operations.length !== 1 ? "s" : ""} planned</span>
        <button class="vibe-select-all" on:click={toggleAll}>
          {selectedOps.every(Boolean) ? "Deselect all" : "Select all"}
        </button>
      </div>

      <div class="vibe-ops">
        {#each mutationPlan.operations as op, i}
          <div class="vibe-op" class:vibe-op-selected={selectedOps[i]} class:vibe-op-executed={executionResults !== null}>
            {#if !executionResults}
              <label class="vibe-op-check">
                <input type="checkbox" bind:checked={selectedOps[i]} />
              </label>
            {/if}
            <div class="vibe-op-badge" style="color:{opColor(op.op)}">
              {opSymbol(op.op)}
            </div>
            <div class="vibe-op-body">
              <div class="vibe-op-top">
                <span class="vibe-op-type" style="color:{opColor(op.op)}">{op.op}</span>
                {#if op.target}
                  <code class="vibe-op-target">{op.target}</code>
                {/if}
              </div>
              <div class="vibe-op-desc">{op.description}</div>
              {#if Object.keys(op.data).length > 0}
                <div class="vibe-op-data">
                  {#each Object.entries(op.data) as [key, value]}
                    <div class="vibe-op-field" style="color:{opColor(op.op)}">
                      <span class="vibe-op-field-key">{key}:</span>
                      <span class="vibe-op-field-val" style="white-space: pre-wrap; word-break: break-word;">{typeof value === "string" ? value : JSON.stringify(value)}</span>
                    </div>
                  {/each}
                </div>
              {/if}
              {#if executionResults && executionResults[i]}
                <div class="vibe-op-result" class:vibe-op-error={executionResults[i].error}>
                  {#if executionResults[i].error}
                    Failed: {executionResults[i].error}
                  {:else}
                    {executionResults[i].result}
                  {/if}
                </div>
              {/if}
            </div>
          </div>
        {/each}
      </div>

      {#if !executionResults}
        <div class="vibe-execute-bar">
          <button
            class="btn-primary vibe-execute-btn"
            on:click={executeApproved}
            disabled={executing || approvedCount === 0}
          >
            {#if executing}
              Executing...
            {:else}
              Execute {approvedCount} operation{approvedCount !== 1 ? "s" : ""}
            {/if}
          </button>
          <span class="vibe-execute-hint">Review the plan above, then execute approved operations.</span>
        </div>
      {:else}
        <div class="vibe-done-bar">
          Mutations applied. {executionResults.filter(r => !r.error).length}/{executionResults.length} succeeded.
        </div>
      {/if}
    {/if}
  {/if}

  <!-- History sidebar -->
  {#if history.length > 0 && !mutationPlan}
    <div class="vibe-history">
      <h3 class="vibe-history-title">Recent</h3>
      {#each history as h}
        <button class="vibe-history-item" on:click={() => { prompt = h.prompt; }}>
          <span class="vibe-history-mode">M</span>
          <span class="vibe-history-prompt">{h.prompt}</span>
        </button>
      {/each}
    </div>
  {/if}

  {#if !loading && !mutationPlan && history.length === 0}
    <div class="vibe-empty">
      <div class="vibe-empty-icon">~</div>
      <p>Type a prompt and press <kbd>Enter</kbd> or click <strong>Plan</strong>.</p>
      <div class="vibe-examples">
        <p class="vibe-examples-title">Try these:</p>
        <button class="vibe-example" on:click={() => { prompt = "Remember that we now use Bun instead of Node"; }}>
          "Remember that we now use Bun instead of Node"
        </button>
        <button class="vibe-example" on:click={() => { prompt = "Mark all testing memories as importance 8"; }}>
          "Mark all testing memories as importance 8"
        </button>
      </div>
    </div>
  {/if}
</section>

<style>
  .vibe-section { display: flex; flex-direction: column; gap: 20px; }

  .vibe-title { font-size: 18px; font-weight: 700; color: var(--text-bold); margin-bottom: 4px; }
  .vibe-desc { font-size: 13px; color: var(--text-muted); }

  .vibe-controls {
    display: flex; flex-direction: column; gap: 16px;
    background: var(--bg-card); border: 1px solid var(--border-main); 
    padding: 24px; box-shadow: var(--shadow-sm);
  }

  .vibe-input-row { display: flex; gap: 16px; align-items: flex-start; }

  .vibe-input {
    flex: 1; background: var(--bg-main); border: 1px solid var(--border-main); color: var(--text-main);
     padding: 16px; font-size: 15px; outline: none;
    font-family: inherit; resize: vertical; min-height: 56px;
  }
  .vibe-input:focus { border-color: var(--accent-main); }
  .vibe-input:disabled { opacity: 0.5; }

  .vibe-input-actions { display: flex; flex-direction: column; gap: 8px; min-width: 160px; }

  .vibe-opts { display: flex; gap: 12px; }
  .vibe-opts label {
    display: flex; flex-direction: column; gap: 4px;
    font-size: 12px; color: var(--text-light); font-weight: 600;
  }
  .vibe-opt-input {
    background: var(--bg-main); border: 1px solid var(--border-main); color: var(--text-main);
     padding: 8px 10px; font-size: 13px; outline: none;
    width: 90px;
  }
  .vibe-opt-num { width: 60px; text-align: center; }

  .vibe-submit { padding: 12px 24px; font-size: 14px; white-space: nowrap; }

  /* Error */
  .vibe-error {
    display: flex; align-items: center; gap: 12px;
    padding: 10px 14px; background: var(--bg-error); color: var(--text-error);
     font-size: 13px;
  }
  .vibe-error button { margin-left: auto; background: none; border: none; color: var(--text-error); cursor: pointer; }

  /* Reasoning */
  .vibe-reasoning {
    background: #1a1a2e; border: 1px solid #2d2b55;
     padding: 12px 14px;
    font-size: 13px; color: var(--text-active); line-height: 1.5;
  }
  .vibe-reasoning-label {
    display: inline-block; font-size: 10px; font-weight: 700;
    text-transform: uppercase; letter-spacing: 0.05em;
    color: var(--accent-main); margin-right: 8px;
    background: var(--bg-active); padding: 2px 7px; 
  }

  /* Mutation plan */
  .vibe-mutation-header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 4px 0;
    font-size: 13px; color: var(--text-secondary);
  }
  .vibe-select-all {
    background: none; border: 1px solid var(--border-main); color: var(--text-secondary);
    padding: 4px 10px;  cursor: pointer; font-size: 12px;
  }
  .vibe-select-all:hover { background: var(--border-main); }

  .vibe-ops { display: flex; flex-direction: column; gap: 12px; }

  .vibe-op {
    display: flex; gap: 16px; align-items: flex-start;
    background: var(--bg-card); border: 1px solid var(--border-main); 
    padding: 16px 20px; transition: all 0.15s;
  }
  .vibe-op-selected { border-color: #4f46e5; background: #1e2640; }
  .vibe-op-executed { opacity: 0.7; }

  .vibe-op-check { display: flex; align-items: center; padding-top: 2px; cursor: pointer; }
  .vibe-op-check input { accent-color: var(--accent-main); width: 16px; height: 16px; cursor: pointer; }

  .vibe-op-badge {
    font-size: 18px; font-weight: 700; font-family: monospace;
    min-width: 20px; text-align: center; padding-top: 1px;
  }

  .vibe-op-body { flex: 1; display: flex; flex-direction: column; gap: 6px; }
  .vibe-op-top { display: flex; align-items: center; gap: 8px; }
  .vibe-op-type { font-size: 12px; font-weight: 700; font-family: monospace; }
  .vibe-op-target { font-size: 11px; color: var(--text-link); }
  .vibe-op-desc { font-size: 13px; color: var(--text-dim); white-space: pre-wrap; word-break: break-word; }

  .vibe-op-data {
    display: flex; flex-direction: column; gap: 2px;
    background: var(--bg-main);  padding: 8px 10px;
    font-family: monospace; font-size: 12px;
  }
  .vibe-op-field { display: flex; gap: 6px; }
  .vibe-op-field-key { color: var(--text-muted); min-width: 80px; }
  .vibe-op-field-val { color: var(--text-secondary); word-break: break-word; }

  .vibe-op-result {
    font-size: 12px; color: var(--text-success);
    background: var(--bg-success); padding: 6px 10px; 
    margin-top: 4px;
  }
  .vibe-op-error { color: var(--text-error); background: #2a0f0f; }

  .vibe-execute-bar {
    display: flex; align-items: center; gap: 16px;
    padding: 12px 0;
  }
  .vibe-execute-btn { padding: 10px 24px; font-size: 14px; }
  .vibe-execute-hint { font-size: 12px; color: var(--text-light); }

  .vibe-done-bar {
    padding: 12px 16px; background: var(--bg-success); border: 1px solid #166534;
     color: var(--text-success); font-size: 13px; font-weight: 500;
  }

  /* History */
  .vibe-history { display: flex; flex-direction: column; gap: 6px; }
  .vibe-history-title {
    font-size: 12px; color: var(--text-light); text-transform: uppercase;
    letter-spacing: 0.05em; margin-bottom: 4px;
  }
  .vibe-history-item {
    display: flex; align-items: center; gap: 10px;
    background: var(--bg-card); border: 1px solid var(--border-main); 
    padding: 8px 12px; cursor: pointer; text-align: left;
    transition: border-color 0.15s; color: inherit;
  }
  .vibe-history-item:hover { border-color: var(--text-light); }
  .vibe-history-mode {
    display: inline-flex; align-items: center; justify-content: center;
    width: 22px; height: 22px; 
    font-size: 11px; font-weight: 700;
    background: var(--bg-active); color: var(--text-active);
  }
  .vibe-history-prompt {
    font-size: 13px; color: var(--text-secondary);
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1;
  }

  /* Empty state */
  .vibe-empty {
    display: flex; flex-direction: column; align-items: center;
    gap: 16px; padding: 48px 20px; text-align: center; color: var(--text-light);
  }
  .vibe-empty-icon { font-size: 48px; color: var(--border-main); font-family: monospace; font-weight: 700; }
  .vibe-empty p { font-size: 14px; }
  .vibe-empty kbd {
    background: var(--bg-card); border: 1px solid var(--border-main); 
    padding: 1px 5px; font-size: 11px; color: var(--text-secondary);
  }

  .vibe-examples {
    display: flex; flex-direction: column; gap: 6px; align-items: center;
    margin-top: 8px;
  }
  .vibe-examples-title { font-size: 12px; color: var(--text-muted); margin-bottom: 4px; }
  .vibe-example {
    background: var(--bg-card); border: 1px solid var(--border-main); 
    padding: 8px 16px; cursor: pointer; color: var(--text-link); font-size: 13px;
    font-style: italic; transition: border-color 0.15s;
  }
  .vibe-example:hover { border-color: var(--accent-main); }
</style>
