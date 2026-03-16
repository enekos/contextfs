<script lang="ts">
  type Row = Record<string, any>;

  // @ts-ignore
  const API_BASE = import.meta.env.VITE_DASHBOARD_API_BASE || "http://localhost:8787";

  let loading = false;
  let error = "";
  let skills: Row[] = [];
  let memories: Row[] = [];
  let contextNodes: Row[] = [];

  let activeTab: "overview" | "skills" | "memories" | "contextNodes" = "overview";
  let searchTerm = "";

  $: filteredSkills = skills.filter(s => 
    !searchTerm || 
    (s.name?.toLowerCase() || "").includes(searchTerm.toLowerCase()) ||
    (s.description?.toLowerCase() || "").includes(searchTerm.toLowerCase())
  );

  $: filteredMemories = memories.filter(m => 
    !searchTerm || 
    (m.content?.toLowerCase() || "").includes(searchTerm.toLowerCase()) ||
    (m.category?.toLowerCase() || "").includes(searchTerm.toLowerCase())
  );

  $: filteredContextNodes = contextNodes.filter(c => 
    !searchTerm || 
    (c.name?.toLowerCase() || "").includes(searchTerm.toLowerCase()) ||
    (c.uri?.toLowerCase() || "").includes(searchTerm.toLowerCase()) ||
    (c.abstract?.toLowerCase() || "").includes(searchTerm.toLowerCase())
  );

  // Form states
  let newSkill = { name: "", description: "", metadata: "" };
  let newMemory = { content: "", category: "core", owner: "user", importance: 5 };
  let newContext = { uri: "", parent_uri: "", name: "", abstract: "", metadata: "" };

  async function loadDashboard() {
    loading = true;
    error = "";
    try {
      const res = await fetch(`${API_BASE}/api/dashboard?limit=200`);
      if (!res.ok) throw new Error(`API error: ${res.status}`);
      const payload = await res.json();
      skills = payload.skills ?? [];
      memories = payload.memories ?? [];
      contextNodes = payload.contextNodes ?? [];
    } catch (err: any) {
      error = err.message || "Unknown error";
    } finally {
      loading = false;
    }
  }

  async function deleteItem(type: "skills" | "memories" | "context", idOrUri: string, idParam = "id") {
    if (!confirm(`Are you sure you want to delete this ${type}?`)) return;
    loading = true;
    try {
      const res = await fetch(`${API_BASE}/api/${type}?${idParam}=${encodeURIComponent(idOrUri)}`, {
        method: "DELETE"
      });
      if (!res.ok) throw new Error(`Failed to delete`);
      await loadDashboard();
    } catch (err: any) {
      error = err.message || "Unknown error";
    } finally {
      loading = false;
    }
  }

  async function createSkill() {
    loading = true;
    try {
      let meta = {};
      try {
        if (newSkill.metadata) meta = JSON.parse(newSkill.metadata);
      } catch (e) {
        throw new Error("Invalid JSON in Metadata");
      }
      
      const res = await fetch(`${API_BASE}/api/skills`, {
        method: "POST",
        body: JSON.stringify({ ...newSkill, metadata: meta })
      });
      if (!res.ok) throw new Error("Failed to create skill");
      newSkill = { name: "", description: "", metadata: "" };
      await loadDashboard();
    } catch (err: any) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  async function createMemory() {
    loading = true;
    try {
      const res = await fetch(`${API_BASE}/api/memories`, {
        method: "POST",
        body: JSON.stringify(newMemory)
      });
      if (!res.ok) throw new Error("Failed to create memory");
      newMemory = { content: "", category: "core", owner: "user", importance: 5 };
      await loadDashboard();
    } catch (err: any) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  async function createContext() {
    loading = true;
    try {
      let meta = {};
      try {
        if (newContext.metadata) meta = JSON.parse(newContext.metadata);
      } catch (e) {
        throw new Error("Invalid JSON in Metadata");
      }

      const res = await fetch(`${API_BASE}/api/context`, {
        method: "POST",
        body: JSON.stringify({ ...newContext, metadata: meta })
      });
      if (!res.ok) throw new Error("Failed to create context node");
      newContext = { uri: "", parent_uri: "", name: "", abstract: "", metadata: "" };
      await loadDashboard();
    } catch (err: any) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  function pretty(value: unknown): string {
    if (value === null || value === undefined) return "";
    if (typeof value === "object") return JSON.stringify(value, null, 2);
    return String(value);
  }

  function copyText(text: string) {
    navigator.clipboard.writeText(text);
  }

  loadDashboard();
</script>

<main>
  <header>
    <div class="header-left">
      <h1>🗄️ Turso Context Dashboard</h1>
      <nav class="tabs">
        <button class:active={activeTab === 'overview'} on:click={() => { activeTab = 'overview'; searchTerm = ''; }}>Overview</button>
        <button class:active={activeTab === 'skills'} on:click={() => { activeTab = 'skills'; searchTerm = ''; }}>Skills ({skills.length})</button>
        <button class:active={activeTab === 'memories'} on:click={() => { activeTab = 'memories'; searchTerm = ''; }}>Memories ({memories.length})</button>
        <button class:active={activeTab === 'contextNodes'} on:click={() => { activeTab = 'contextNodes'; searchTerm = ''; }}>Context Nodes ({contextNodes.length})</button>
      </nav>
    </div>
    <div class="header-right" style="display: flex; gap: 12px; align-items: center;">
      {#if activeTab !== 'overview'}
        <input type="text" placeholder="Search..." bind:value={searchTerm} style="width: 200px; padding: 6px 12px;" />
      {/if}
      <button class="btn-primary" on:click={loadDashboard} disabled={loading}>
        {loading ? "⏳ Refreshing..." : "🔄 Refresh"}
      </button>
    </div>
  </header>

  {#if error}
    <div class="error-banner">
      <strong>Error:</strong> {error}
      <button class="close-error" on:click={() => error = ""}>×</button>
    </div>
  {/if}

  <div class="content">
    {#if activeTab === 'overview'}
      <section class="cards">
        <article class="card" on:click={() => activeTab = 'skills'} role="button" tabindex="0">
          <div class="card-icon">🧠</div>
          <div class="card-content">
            <h2>Skills</h2>
            <p>{skills.length}</p>
          </div>
        </article>
        <article class="card" on:click={() => activeTab = 'memories'} role="button" tabindex="0">
          <div class="card-icon">💾</div>
          <div class="card-content">
            <h2>Memories</h2>
            <p>{memories.length}</p>
          </div>
        </article>
        <article class="card" on:click={() => activeTab = 'contextNodes'} role="button" tabindex="0">
          <div class="card-icon">📁</div>
          <div class="card-content">
            <h2>Context Nodes</h2>
            <p>{contextNodes.length}</p>
          </div>
        </article>
      </section>

    {:else if activeTab === 'skills'}
      <section class="action-panel">
        <h3>Add New Skill</h3>
        <form on:submit|preventDefault={createSkill} class="inline-form">
          <input type="text" placeholder="Name" bind:value={newSkill.name} required />
          <input type="text" placeholder="Description" bind:value={newSkill.description} required />
          <input type="text" placeholder='Metadata (JSON)' bind:value={newSkill.metadata} />
          <button type="submit" class="btn-success" disabled={loading}>+ Add Skill</button>
        </form>
      </section>

      <section class="table-wrap">
        <table>
          <thead>
            <tr><th>ID</th><th>Name</th><th>Description</th><th>Metadata</th><th>Created</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {#if filteredSkills.length === 0}
              <tr><td colspan="6" class="empty-state">No skills found</td></tr>
            {/if}
            {#each filteredSkills as row}
              <tr>
                <td class="id-cell" title="Copy {row.id}" style="cursor:pointer;" on:click={() => copyText(row.id)}>{row.id.substring(0,8)}...</td>
                <td class="fw-bold">{pretty(row.name)}</td>
                <td>{pretty(row.description)}</td>
                <td><pre class="json-cell">{pretty(row.metadata)}</pre></td>
                <td class="date-cell">{new Date(row.created_at).toLocaleString()}</td>
                <td><button class="btn-danger btn-sm" on:click={() => deleteItem('skills', row.id)}>Delete</button></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>

    {:else if activeTab === 'memories'}
      <section class="action-panel">
        <h3>Add New Memory</h3>
        <form on:submit|preventDefault={createMemory} class="inline-form">
          <input type="text" placeholder="Content" bind:value={newMemory.content} required />
          <input type="text" placeholder="Category" bind:value={newMemory.category} />
          <input type="text" placeholder="Owner" bind:value={newMemory.owner} />
          <input type="number" placeholder="Importance (1-10)" bind:value={newMemory.importance} min="1" max="10" />
          <button type="submit" class="btn-success" disabled={loading}>+ Add Memory</button>
        </form>
      </section>

      <section class="table-wrap">
        <table>
          <thead>
            <tr><th>ID</th><th>Content</th><th>Category</th><th>Owner</th><th>Importance</th><th>Created</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {#if filteredMemories.length === 0}
              <tr><td colspan="7" class="empty-state">No memories found</td></tr>
            {/if}
            {#each filteredMemories as row}
              <tr>
                <td class="id-cell" title="Copy {row.id}" style="cursor:pointer;" on:click={() => copyText(row.id)}>{row.id.substring(0,8)}...</td>
                <td>{pretty(row.content)}</td>
                <td><span class="badge">{pretty(row.category)}</span></td>
                <td>{pretty(row.owner)}</td>
                <td><span class="importance-badge i-{row.importance}">{pretty(row.importance)}</span></td>
                <td class="date-cell">{new Date(row.created_at).toLocaleString()}</td>
                <td><button class="btn-danger btn-sm" on:click={() => deleteItem('memories', row.id)}>Delete</button></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>

    {:else if activeTab === 'contextNodes'}
      <section class="action-panel">
        <h3>Add New Context Node</h3>
        <form on:submit|preventDefault={createContext} class="grid-form">
          <input type="text" placeholder="URI (e.g. file:///src)" bind:value={newContext.uri} required />
          <input type="text" placeholder="Parent URI" bind:value={newContext.parent_uri} />
          <input type="text" placeholder="Name" bind:value={newContext.name} required />
          <input type="text" placeholder="Abstract / Summary" bind:value={newContext.abstract} />
          <input type="text" placeholder='Metadata (JSON)' bind:value={newContext.metadata} />
          <button type="submit" class="btn-success" disabled={loading}>+ Add Context Node</button>
        </form>
      </section>

      <section class="table-wrap">
        <table>
          <thead>
            <tr><th>URI</th><th>Parent</th><th>Name</th><th>Abstract</th><th>Created</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {#if filteredContextNodes.length === 0}
              <tr><td colspan="6" class="empty-state">No context nodes found</td></tr>
            {/if}
            {#each filteredContextNodes as row}
              <tr>
                <td class="fw-bold" style="cursor:pointer;" title="Copy URI" on:click={() => copyText(row.uri)}>{pretty(row.uri)}</td>
                <td class="text-muted">{pretty(row.parent_uri) || '-'}</td>
                <td>{pretty(row.name)}</td>
                <td>{pretty(row.abstract)}</td>
                <td class="date-cell">{new Date(row.created_at).toLocaleString()}</td>
                <td><button class="btn-danger btn-sm" on:click={() => deleteItem('context', row.uri, 'uri')}>Delete</button></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>
    {/if}
  </div>
</main>
