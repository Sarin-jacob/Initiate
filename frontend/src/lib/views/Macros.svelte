<script>
    import { onMount } from 'svelte';

    let macros = [];
    let servers = [];
    
    // Dynamically populated from connected agents: { "system_user": ["create", "set_password"], ... }
    let capabilityMap = {}; 

    // Form State
    let formName = '';
    let formDesc = '';
    let formSteps = [];
    let selectedModule = '';
    let selectedAction = '';

    let isSaving = false;
    let alertMsg = '';
    let isError = false;

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
        } catch (err) {
            console.error("Failed to load data", err);
        }
    }

    // Extracts and merges capabilities from all known agents
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
        
        // Convert Sets back to standard arrays for Svelte iteration
        for (const mod in map) {
            capabilityMap[mod] = Array.from(map[mod]);
        }
    }

    onMount(fetchData);

    // Pipeline Sequence Controls
    function addStep() {
        if (!selectedModule || !selectedAction) return;
        formSteps = [...formSteps, { module: selectedModule, action: selectedAction }];
        selectedAction = ''; // Reset action for quick chaining
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

    async function handleSave(e) {
        e.preventDefault();
        isSaving = true;
        try {
            const res = await fetch('/api/admin/macros', {
                method: 'POST',
                headers,
                body: JSON.stringify({ name: formName, description: formDesc, steps: formSteps })
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to save Macro");
            
            alertMsg = "Pipeline Macro created successfully!";
            isError = false;
            
            // Reset form
            formName = ''; formDesc = ''; formSteps = [];
            fetchData();
        } catch (err) {
            alertMsg = err.message;
            isError = true;
        } finally {
            isSaving = false;
        }
    }
</script>

<div class="space-y-8">
    <div>
        <h1 class="text-4xl font-bold">Provisioning Macros</h1>
        <p class="text-base-content/70 mt-2 text-lg">Build and manage sequential execution pipelines for Edge Agents.</p>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
        
        <!-- MACRO BUILDER -->
        <div class="card bg-base-100 shadow-sm border border-base-300">
            <div class="card-body">
                <h2 class="card-title text-xl border-b border-base-200 pb-4 mb-2">Create New Pipeline</h2>
                
                {#if alertMsg}
                    <div class="alert {isError ? 'alert-error' : 'alert-success'} shadow-sm mb-4">{alertMsg}</div>
                {/if}

                <form on:submit={handleSave} class="space-y-6">
                    <div class="space-y-4">
                        <input type="text" bind:value={formName} required class="input input-bordered w-full text-lg font-bold" placeholder="Macro Name (e.g., Standard Linux Dev)" />
                        <input type="text" bind:value={formDesc} required class="input input-bordered w-full" placeholder="Short description..." />
                    </div>

                    <div class="bg-base-200/50 p-4 rounded-xl border border-base-300">
                        <h3 class="font-bold text-sm uppercase tracking-wider opacity-60 mb-4">Execution Sequence</h3>
                        
                        <!-- The Ordered Pipeline Blocks -->
                        <div class="space-y-2 mb-6">
                            {#each formSteps as step, i}
                                <div class="flex items-center gap-3 bg-base-100 p-3 rounded-lg border border-base-300 shadow-sm transition-all">
                                    <div class="flex flex-col gap-1">
                                        <button type="button" class="btn btn-xs btn-ghost p-1 h-auto min-h-0" disabled={i === 0} on:click={() => moveStep(i, -1)}>▲</button>
                                        <button type="button" class="btn btn-xs btn-ghost p-1 h-auto min-h-0" disabled={i === formSteps.length - 1} on:click={() => moveStep(i, 1)}>▼</button>
                                    </div>
                                    <div class="flex-1 font-mono text-sm">
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
                        <div class="flex gap-2">
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
                        Save Macro Pipeline
                    </button>
                </form>
            </div>
        </div>

        <!-- EXISTING MACROS -->
        <div class="card bg-base-100 shadow-sm border border-base-300">
            <div class="card-body">
                <h2 class="card-title text-xl border-b border-base-200 pb-4 mb-4">Saved Pipelines</h2>
                
                <div class="space-y-4 overflow-y-auto max-h-[600px] pr-2">
                    {#each macros as macro}
                        <div class="border border-base-200 rounded-xl p-4 bg-base-200/20 hover:bg-base-200/50 transition-colors">
                            <div class="font-bold text-lg">{macro.Name}</div>
                            <div class="text-sm opacity-70 mb-4">{macro.Description}</div>
                            
                            <div class="flex flex-wrap gap-2">
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