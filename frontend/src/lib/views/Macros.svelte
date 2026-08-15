<script>
    import { onMount } from 'svelte';
    import { addToast } from '../stores/toast.js';

    let macros = [];
    let servers = [];
    let capabilityMap = {}; 

    let editingId = null; 
    let formName = '';
    let formDesc = '';
    let formSteps = [];
    
    let selectedModule = '';
    let selectedAction = '';
    let isSaving = false;

    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + sessionStorage.getItem('nexus_jwt') };

    onMount(fetchData);

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
                    if (!map[mod]) map[mod] = {};
                    for (const [act, vars] of Object.entries(actions)) {
                        map[mod][act] = vars;
                    }
                }
            } catch(e) {}
        });
        capabilityMap = map;
    }

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
            const parsedSteps = typeof macro.Steps === 'string' ? JSON.parse(macro.Steps) : 
                                (typeof macro.steps === 'string' ? JSON.parse(macro.steps) : (macro.Steps || macro.steps || [])); 
            
            // DECOMPILE BACK TO UI STATE (Now using safe arrays)
            formSteps = parsedSteps.map(step => {
                const decompiledParams = [];
                const reqVars = (capabilityMap[step.module] || {})[step.action] || {};

                for (const [varName, compiledVal] of Object.entries(step.params || {})) {
                    let source = 'static';
                    let value = compiledVal;
                    let type = reqVars[varName] || 'string'; // Look up exact type

                    if (typeof compiledVal === 'string' && compiledVal.startsWith('{{') && compiledVal.endsWith('}}')) {
                        const inner = compiledVal.slice(2, -2);
                        const parts = inner.split('.');
                        source = parts[0];
                        value = parts.slice(1).join('.');
                    }
                    
                    decompiledParams.push({ name: varName, source, value, type });
                }
                return { ...step, params: decompiledParams };
            });
        } catch { 
            formSteps = []; 
        }
    }

    // --- Dynamic Step Builder ---
    function addStep() {
        if (!selectedModule || !selectedAction) return;
        
        const requiredVars = capabilityMap[selectedModule][selectedAction] || {};
        const stepParams = [];
        
        for (const varName in requiredVars) {
            let defaultSource = 'static';
            let defaultValue = '';
            let type = requiredVars[varName];
            
            if (type === 'secret') {
                defaultSource = 'user'; defaultValue = varName;
            } else if (type === 'textarea') {
                defaultSource = 'user'; defaultValue = varName;
            } else if (varName.includes('username')) {
                defaultSource = 'sys'; defaultValue = 'username';
            } else if (type === 'bool' || type === 'boolean') {
                defaultValue = 'false';
            }
            
            stepParams.push({ 
                name: varName, 
                source: defaultSource, 
                value: defaultValue,
                type: type 
            });
        }

        formSteps = [...formSteps, { module: selectedModule, action: selectedAction, params: stepParams }];
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
        
        // Re-compile the array back into the mapping string
        const compiledSteps = formSteps.map(step => {
            const compiledParams = {};
            for (const param of step.params || []) {
                if (param.source === 'static') {
                    compiledParams[param.name] = param.value;
                } else {
                    compiledParams[param.name] = `{{${param.source}.${param.value}}}`;
                }
            }
            return { module: step.module, action: step.action, params: compiledParams };
        });

        const url = editingId ? `/api/admin/macros/${editingId}` : '/api/admin/macros';
        const method = editingId ? 'PUT' : 'POST';

        try {
            const res = await fetch(url, {
                method, headers,
                body: JSON.stringify({ name: formName, description: formDesc, steps: compiledSteps })
            });
            if (!res.ok) throw new Error("Failed to save Macro");
            
            addToast(`Macro saved successfully!`, "success");
            resetForm();
            fetchData();
        } catch (err) { addToast(err.message, "error"); } 
        finally { isSaving = false; }
    }

    async function handleDelete(id, name) {
        if (!confirm(`Are you sure you want to delete Macro "${name}"?`)) return;
        try {
            const res = await fetch(`/api/admin/macros/${id}`, { method: 'DELETE', headers });
            if (!res.ok) throw new Error("Failed to delete");
            addToast("Macro deleted.", "success");
            if (editingId === id) resetForm();
            fetchData();
        } catch (err) { addToast(err.message, "error"); }
    }
</script>

<div class="space-y-8">
    <div>
        <h1 class="text-4xl font-bold">Parameter-Bound Pipelines</h1>
        <p class="text-base-content/70 mt-2 text-lg">Map system context and user inputs to edge capabilities.</p>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-8">
        <div class="card bg-base-100 shadow-sm border border-base-300">
            <div class="card-body">
                <div class="flex justify-between border-b border-base-200 pb-4 mb-2">
                    <h2 class="card-title text-xl">{editingId ? 'Update Macro' : 'Create New Macro'}</h2>
                    {#if editingId}<button class="btn btn-sm btn-ghost" on:click={resetForm}>Clear / New</button>{/if}
                </div>

                <form on:submit={handleSave} class="space-y-6">
                    <div class="space-y-4">
                        <input type="text" bind:value={formName} required class="input input-bordered w-full text-lg font-bold" placeholder="Macro Name" />
                        <input type="text" bind:value={formDesc} required class="input input-bordered w-full" placeholder="Short description..." />
                    </div>

                    <div class="bg-base-200/50 p-4 rounded-xl border border-base-300">
                        <h3 class="font-bold text-sm uppercase tracking-wider opacity-60 mb-4">Execution Sequence</h3>
                        
                        <div class="space-y-4 mb-6">
                            {#each formSteps as step, i}
                                <div class="bg-base-100 p-4 rounded-lg border border-base-300 shadow-sm">
                                    <div class="flex justify-between items-center mb-4 pb-2 border-b border-base-200">
                                        <div class="font-mono text-sm font-bold">
                                            <span class="opacity-50 mr-2">{i + 1}.</span>
                                            <span class="text-primary">{step.module}</span>:<span class="text-secondary">{step.action}</span>
                                        </div>
                                        <div class="flex gap-2">
                                            <button type="button" class="btn btn-xs" disabled={i === 0} on:click={() => moveStep(i, -1)}>↑</button>
                                            <button type="button" class="btn btn-xs" disabled={i === formSteps.length - 1} on:click={() => moveStep(i, 1)}>↓</button>
                                            <button type="button" class="btn btn-xs btn-error btn-outline" on:click={() => removeStep(i)}>✕</button>
                                        </div>
                                    </div>
                                    
                                    <div class="space-y-3">
                                        {#if step.params.length === 0}
                                            <div class="text-xs opacity-50">No parameters required for this action.</div>
                                        {/if}
                                        
                                        {#each step.params as param}
                                            <div class="flex flex-col md:flex-row gap-2 items-start md:items-center">
                                                <div class="font-mono text-xs w-32 truncate" title={param.name}>{param.name}</div>
                                                
                                                <select 
                                                    bind:value={param.source} 
                                                    on:change={() => param.value = param.source === 'sys' ? 'username' : ((param.type === 'bool' || param.type === 'boolean') ? 'false' : '')} 
                                                    class="select select-bordered select-xs w-full md:w-32"
                                                >
                                                    <option value="sys">System Data</option>
                                                    <option value="admin">Admin Prompt</option>
                                                    <option value="user">User Prompt</option>
                                                    <option value="static">Static Value</option>
                                                </select>
                                                
                                                {#if param.source === 'static'}
                                                    {#if param.type === 'secret'}
                                                        <input type="password" bind:value={param.value} class="input input-bordered input-xs w-full font-mono" placeholder="Secret..." />
                                                    {:else if param.type === 'textarea'}
                                                        <textarea bind:value={param.value} class="textarea textarea-bordered textarea-xs w-full font-mono leading-tight" rows="1" placeholder="Data..."></textarea>
                                                    {:else if param.type === 'bool' || param.type === 'boolean'}
                                                        <select bind:value={param.value} class="select select-bordered select-xs w-full font-mono">
                                                            <option value="true">True</option>
                                                            <option value="false">False</option>
                                                        </select>
                                                    {:else if param.type === 'int' || param.type === 'number'}
                                                        <input type="number" bind:value={param.value} class="input input-bordered input-xs w-full font-mono" />
                                                    {:else}
                                                        <input type="text" bind:value={param.value} class="input input-bordered input-xs w-full font-mono" placeholder="Hardcoded string" />
                                                    {/if}

                                                {:else if param.source === 'sys'}
                                                    <select bind:value={param.value} class="select select-bordered select-xs w-full font-mono">
                                                        <option value="username">username</option>
                                                        <option value="email">email</option>
                                                        <option value="id">user_id</option>
                                                    </select>

                                                {:else}
                                                    <input type="text" bind:value={param.value}
                                                    required
                                                    on:input={(e) => param.value = e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '')}
                                                    class="input input-bordered input-xs w-full font-mono" placeholder="Prompt variable name..." />
                                                {/if}
                                            </div>
                                        {/each}
                                    </div>
                                </div>
                            {:else}
                                <div class="text-center p-6 border-2 border-dashed border-base-300 rounded-lg text-sm opacity-50">Macro is empty.</div>
                            {/each}
                        </div>

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
                                    {#each Object.keys(capabilityMap[selectedModule]) as act}
                                        <option value={act}>{act}</option>
                                    {/each}
                                {/if}
                            </select>
                            
                            <button type="button" class="btn btn-sm btn-neutral" on:click={addStep} disabled={!selectedModule || !selectedAction}>Add Step</button>
                        </div>
                    </div>

                    <button type="submit" class="btn btn-primary w-full" disabled={isSaving || formSteps.length === 0}>
                        {#if isSaving} <span class="loading loading-spinner"></span> {/if}
                        {editingId ? 'Save Macro' : 'Create Macro'}
                    </button>
                </form>
            </div>
        </div>

        <div class="card bg-base-100 shadow-sm border border-base-300">
            <div class="card-body">
                <h2 class="card-title text-xl mb-4">Saved Pipelines</h2>
                <div class="space-y-4 overflow-y-auto max-h-[700px]">
                    {#each macros as macro}
                        <div class="border border-base-200 rounded-xl p-4 bg-base-200/20 hover:bg-base-200/60 group relative cursor-pointer" on:click={() => selectMacro(macro)}>
                            <div class="font-bold">{macro.Name}</div>
                            <div class="text-sm opacity-70 mb-2">{macro.Description}</div>
                            <div class="absolute top-0 right-0 opacity-0 group-hover:opacity-100 transition-opacity m-4 z-10">
                                <button type="button" class="btn btn-xs btn-outline btn-error" on:click|stopPropagation={() => handleDelete(macro.ID, macro.Name)}>Delete</button>
                            </div>
                            <div class="flex flex-wrap gap-1">
                                {#each (typeof macro.Steps === 'string' ? JSON.parse(macro.Steps) : (macro.Steps || macro.steps || [])) as step, i}
                                    <span class="badge badge-outline badge-sm font-mono opacity-70">{i+1}. {step.module}:{step.action}</span>
                                {/each}
                            </div>
                        </div>
                    {/each}
                </div>
            </div>
        </div>

    </div>
</div>