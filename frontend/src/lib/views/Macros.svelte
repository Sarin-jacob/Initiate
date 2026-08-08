<script>
    import { onMount } from 'svelte';
    import { addToast } from '../stores/toast.js';

    let macros = [];
    let servers = [];
    let capabilityMap = {}; 

    // Form State
    let editingId = null; // Tracks if we are Creating or Updating
    let formName = '';
    let formDesc = '';
    let formSteps = [];
    let selectedModule = '';
    let selectedAction = '';

    let isSaving = false;

    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchData() {
        try {
            const [macRes, srvRes] = await Promise.all([
                fetch('/api/admin/macros', { headers }),
                fetch('/api/admin/servers', { headers })
            ]);
            if (macRes.ok) macros = await macRes.json() || [];
            if (srvRes.ok) {
                servers = await srvRes.json() || [];
                buildCapabilityMap(servers);
            }
        } catch (err) { addToast("Failed to load macros", "error"); }
    }

    function buildCapabilityMap(srvList) {
        const map = {};
        srvList.forEach(s => {
            if (!s.Capabilities) return;
            try {
                const caps = JSON.parse(s.Capabilities);
                for (const [mod, actions] of Object.entries(caps)) {
                    if (!map[mod]) map[mod] = new Set();
                    actions.forEach(a => map[mod].add(a));
                }
            } catch(e) {}
        });
        for (const mod in map) {
            capabilityMap[mod] = Array.from(map[mod]);
        }
    }

    onMount(fetchData);

    // --- Editor Controls ---
    
    function resetForm() {
        editingId = null;
        formName = '';
        formDesc = '';
        formSteps = [];
        selectedModule = '';
        selectedAction = '';
    }

    function selectMacro(macro) {
        editingId = macro.ID;
        formName = macro.Name;
        formDesc = macro.Description;
        try {
            formSteps = JSON.parse(macro.Steps) || [];
        } catch {
            formSteps = [];
        }
    }

    // --- Pipeline Sequence Math ---

    function addStep() {
        if (!selectedModule || !selectedAction) return;
        formSteps = [...formSteps, { module: selectedModule, action: selectedAction }];
        selectedAction = ''; 
    }

    function removeStep(index) {
        formSteps = formSteps.filter((_, i) => i !== index);
    }

    function moveStep(index, direction) {
        if (index + direction < 0 || index + direction >= formSteps.length) return;
        const newSteps = [...formSteps];
        const temp = newSteps[index];
        newSteps[index] = newSteps[index + direction];
        newSteps[index + direction] = temp;
        formSteps = newSteps;
    }

    // --- API Handlers ---

    async function handleSave(e) {
        e.preventDefault();
        isSaving = true;
        
        const url = editingId ? `/api/admin/macros/${editingId}` : '/api/admin/macros';
        const method = editingId ? 'PUT' : 'POST';

        try {
            const res = await fetch(url, {
                method, headers,
                body: JSON.stringify({ name: formName, description: formDesc, steps: formSteps })
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to save pipeline");
            
            addToast(`Pipeline ${editingId ? 'updated' : 'created'} successfully!`, "success");
            resetForm();
            fetchData();
        } catch (err) { addToast(err.message, "error"); } 
        finally { isSaving = false; }
    }

    async function handleDelete(id, name) {
        if (!confirm(`Are you sure you want to permanently delete the pipeline "${name}"?`)) return;
        try {
            const res = await fetch(`/api/admin/macros/${id}`, { method: 'DELETE', headers });
            if (!res.ok) throw new Error("Failed to delete pipeline");
            
            addToast("Pipeline deleted.", "success");
            if (editingId === id) resetForm();
            fetchData();
        } catch (err) { addToast(err.message, "error"); }
    }
</script>

<div class="space-y-8">
    <div>
        <h1 class="text-4xl font-bold">Provisioning Macros</h1>
        <p class="text-base-content/70 mt-2 text-lg">Build and manage sequential execution pipelines.</p>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-8">
        
        <!-- MACRO BUILDER -->
        <div class="card bg-base-100 shadow-sm border border-base-300">
            <div class="card-body">
                <div class="flex justify-between border-b border-base-200 pb-4 mb-2">
                    <h2 class="card-title text-xl">{editingId ? 'Update Pipeline' : 'Create New Pipeline'}</h2>
                    {#if editingId}
                        <button class="btn btn-sm btn-ghost" on:click={resetForm}>Clear / New</button>
                    {/if}
                </div>

                <form on:submit={handleSave} class="space-y-6">
                    <div class="space-y-4">
                        <input type="text" bind:value={formName} required class="input input-bordered w-full text-lg font-bold" placeholder="Macro Name (e.g., Standard Linux Dev)" />
                        <input type="text" bind:value={formDesc} required class="input input-bordered w-full" placeholder="Short description..." />
                    </div>

                    <div class="bg-base-200/50 p-4 rounded-xl border border-base-300">
                        <h3 class="font-bold text-sm uppercase tracking-wider opacity-60 mb-4">Execution Sequence</h3>
                        
                        <div class="space-y-2 mb-6">
                            {#each formSteps as step, i}
                                <div class="flex items-center gap-3 bg-base-100 p-2 rounded-lg border border-base-300 shadow-sm">
                                    <div class="flex flex-col">
                                        <button type="button" class="btn btn-xs btn-ghost p-1 h-auto min-h-0" disabled={i === 0} on:click={() => moveStep(i, -1)}>
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" /></svg>
                                        </button>
                                        <button type="button" class="btn btn-xs btn-ghost p-1 h-auto min-h-0" disabled={i === formSteps.length - 1} on:click={() => moveStep(i, 1)}>
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
                                        </button>
                                    </div>
                                    <div class="flex-1 font-mono text-sm pl-2">
                                        <span class="opacity-50 mr-2">{i + 1}.</span>
                                        <span class="text-primary font-bold">{step.module}</span>
                                        <span class="opacity-50 mx-1">:</span>
                                        <span class="text-secondary">{step.action}</span>
                                    </div>
                                    <button type="button" class="btn btn-sm btn-circle btn-ghost text-error" on:click={() => removeStep(i)}>✕</button>
                                </div>
                            {:else}
                                <div class="text-center p-6 border-2 border-dashed border-base-300 rounded-lg text-base-content/50 text-sm">
                                    Pipeline is empty. Add execution steps below.
                                </div>
                            {/each}
                        </div>

                        <!-- Add Block Selector -->
                        <div class="flex flex-col md:flex-row gap-2">
                            <select bind:value={selectedModule} class="select select-bordered flex-1 select-sm font-mono" on:change={() => selectedAction = ''}>
                                <option value="" disabled selected>Select Module</option>
                                {#each Object.keys(capabilityMap) as mod}
                                    <option value={mod}>{mod}</option>
                                {/each}
                            </select>
                            
                            <select bind:value={selectedAction} class="select select-bordered flex-1 select-sm font-mono" disabled={!selectedModule}>
                                <option value="" disabled selected>Select Action</option>
                                {#if selectedModule && capabilityMap[selectedModule]}
                                    {#each capabilityMap[selectedModule] as act}
                                        <option value={act}>{act}</option>
                                    {/each}
                                {/if}
                            </select>
                            
                            <button type="button" class="btn btn-sm btn-neutral" on:click={addStep} disabled={!selectedModule || !selectedAction}>Add Step</button>
                        </div>
                    </div>

                    <button type="submit" class="btn btn-primary w-full text-lg" disabled={isSaving || formSteps.length === 0}>
                        {#if isSaving} <span class="loading loading-spinner"></span> {/if}
                        {editingId ? 'Save Changes' : 'Create Pipeline'}
                    </button>
                </form>
            </div>
        </div>

        <!-- EXISTING MACROS -->
        <div class="card bg-base-100 shadow-sm border border-base-300">
            <div class="card-body">
                <h2 class="card-title text-xl border-b border-base-200 pb-4 mb-4">Saved Pipelines</h2>
                
                <div class="space-y-4 overflow-y-auto max-h-[700px] pr-2">
                    {#each macros as macro}
                        <div class="border border-base-200 rounded-xl p-4 bg-base-200/20 hover:bg-base-200/60 transition-colors flex flex-col group {editingId === macro.ID ? 'border-primary bg-primary/5' : ''}">
                            <div class="flex justify-between items-start mb-2">
                                <div>
                                    <div class="font-bold text-lg">{macro.Name}</div>
                                    <div class="text-sm opacity-70">{macro.Description}</div>
                                </div>
                                <div class="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                    <button class="btn btn-xs btn-outline" on:click={() => selectMacro(macro)}>Edit</button>
                                    <button class="btn btn-xs btn-outline btn-error" on:click={() => handleDelete(macro.ID, macro.Name)}>Delete</button>
                                </div>
                            </div>
                            
                            <div class="flex flex-wrap gap-2 mt-2">
                                {#each JSON.parse(macro.Steps) as step, index}
                                    <div class="badge badge-outline badge-md font-mono flex gap-1 items-center bg-base-100">
                                        <span class="opacity-50 text-xs">{index + 1}.</span>
                                        {step.module}:{step.action}
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {:else}
                        <div class="text-center p-8 opacity-50">No macros defined yet.</div>
                    {/each}
                </div>
            </div>
        </div>

    </div>
</div>