<script>
    import { createEventDispatcher } from 'svelte';
    
    export let servers = [];
    export let macros = [];
    export let pages = [];

    const dispatch = createEventDispatcher();
    const headers = { 
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + localStorage.getItem('nexus_jwt') 
                };

    let isInviting = false;
    let alertMsg = '';
    
    let selectedTargets = [];
    let selectedDocs = [];
    let allocations = {};

    let requiredAdminVars = [];
    let adminInputs = {};
    let isInspectingVars = false;

    // Initialize allocations when servers prop loads
    $: if (servers.length > 0 && Object.keys(allocations).length === 0) {
        let initAlloc = {};
        servers.forEach(srv => { initAlloc[srv.ID] = { selected: false, provId: '', deprovId: '' }; });
        allocations = initAlloc;
    }

    function toggleTarget(id) {
        if (selectedTargets.includes(id)) {
            selectedTargets = selectedTargets.filter(t => t !== id);
        } else {
            selectedTargets = [...selectedTargets, id];
        }
        updateAdminVars(selectedTargets);
    }

    function toggleDoc(slug) {
        if (selectedDocs.includes(slug)) {
            selectedDocs = selectedDocs.filter(d => d !== slug);
        } else {
            selectedDocs = [...selectedDocs, slug];
        }
    }

    async function updateAdminVars(targets) {
        if (!targets || targets.length === 0) {
            requiredAdminVars = [];
            adminInputs = {};
            return;
        }

        isInspectingVars = true;
        try {
            const res = await fetch('/api/admin/macros/admin-vars', {
                method: 'POST',
                headers: headers, // Make sure 'headers' is defined at the top of your script!
                body: JSON.stringify({ target_ids: targets }) // Send the fresh array
            });
            const data = await res.json();
            
            const oldInputs = { ...adminInputs };
            adminInputs = {};
            
            requiredAdminVars = data.admin_vars || [];
            requiredAdminVars.forEach(v => {
                adminInputs[v] = oldInputs[v] || '';
            });
        } catch (err) {
            console.error("Failed to inspect macro variables", err);
        } finally {
            isInspectingVars = false;
        }
    }

    async function handleInvite(e) {
        e.preventDefault();
        if (selectedTargets.length === 0) return alertMsg = "Select at least one target system.";
        
        isInviting = true;
        const form = e.target;
        
        const payload = {
            username: form.username.value,
            email: form.email.value,
            expire_amount: parseInt(form.expireAmount.value) || 0,
            expire_unit: form.expireUnit.value,
            target_ids: selectedTargets,
            doc_slugs: selectedDocs,
            admin_inputs: adminInputs
        };

        try {
            const res = await fetch('/api/admin/users/invite', { method: 'POST', headers, body: JSON.stringify(payload) });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to invite user");
            
            alertMsg = "User invited successfully!";
            form.reset();
            selectedTargets = [];
            selectedDocs = [];
            dispatch('refresh'); // Tell parent to reload table data
            
            setTimeout(() => alertMsg = '', 4000);
        } catch (err) { 
            alertMsg = err.message; 
        } finally { 
            isInviting = false; 
        }
    }
</script>

<div class="collapse collapse-arrow bg-base-100 border border-base-300 shadow-sm rounded-t-2xl">
    <input type="checkbox" /> 
    <div class="collapse-title text-lg font-bold p-4 bg-primary rounded-t-2xl text-primary-content">+ Provision New User Access</div>
    <div class="collapse-content border-t border-base-200 p-4">
        <form on:submit={handleInvite} class="space-y-6 pt-4">
            {#if alertMsg}
                <div class="alert {alertMsg.includes('must select') || alertMsg.includes('Failed') ? 'alert-error' : 'alert-success'} shadow-sm mb-4">
                    {alertMsg}
                </div>
            {/if}
            
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="form-control">
                    <label class="label"><span class="label-text font-bold">Username</span></label>
                    <input type="text" name="username" required class="input input-bordered input-lg" />
                </div>
                <div class="form-control">
                    <label class="label"><span class="label-text font-bold">Email Address</span></label>
                    <input type="email" name="email" required class="input input-bordered input-lg" />
                </div>
            </div>
            
            <div class="form-control max-w-md">
                <label class="label"><span class="label-text font-bold">Automated Expiration</span></label>
                <div class="join w-full">
                    <input type="number" name="expireAmount" min="0" placeholder="0 = Never" class="input input-bordered join-item w-full input-lg" />
                    <select name="expireUnit" class="select select-bordered text-primary-content join-item bg-primary input-lg h-auto w-1/3">
                        <option value="days">Days</option>
                        <option value="weeks">Weeks</option>
                        <option value="months">Months</option>
                        <option value="years">Years</option>
                    </select>
                </div>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 mt-8 border-t border-base-200 pt-8">
                <!-- TARGET SERVER CHECKBOXES -->
                <div class="form-control mb-4">
                    <label class="label"><span class="label-text font-bold">Assign Edge Servers</span></label>
                    <div class="space-y-2 border border-base-300 p-4 rounded-xl max-h-48 overflow-y-auto">
                        {#each servers as server}
                            <label class="flex items-center gap-4 p-4 border border-base-300 rounded-xl cursor-pointer hover:bg-base-200/50 transition-colors {selectedTargets.includes(server.ID) ? 'border-primary bg-primary/5' : 'bg-base-100'}">
                                <input type="checkbox" class="checkbox checkbox-primary" checked={selectedTargets.includes(server.ID)} on:change={() => toggleTarget(server.ID)} />
                                <div>
                                    <div class="font-bold">{server.Name}</div>
                                    <div class="text-sm opacity-60">
                                        {server.ID === 'internal-gitea' ? 'Virtual System' : 'Edge Agent ' + server.ID.substring(0,8)}
                                    </div>
                                </div>
                            </label>
                        {/each}
                    </div>
                </div>

                <!-- DYNAMIC ADMIN PROMPTS -->
                {#if isInspectingVars}
                    <div class="flex items-center gap-2 text-sm opacity-50 mb-4">
                        <span class="loading loading-spinner loading-xs"></span> Inspecting pipelines...
                    </div>
                {:else if requiredAdminVars.length > 0}
                    <div class="bg-base-200/50 p-4 rounded-xl border border-base-300 mb-6 space-y-4">
                        <h3 class="text-sm font-bold opacity-70 flex items-center gap-2">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0..."></path></svg>
                            Required Pipeline Context
                        </h3>
                        
                        {#each requiredAdminVars as varName}
                            <div class="form-control">
                                <label class="label"><span class="label-text font-bold capitalize">{varName.replace(/_/g, ' ')}</span></label>
                                <input type="text" bind:value={adminInputs[varName]} required class="input input-bordered input-sm w-full font-mono" placeholder="Provide value for {varName}..." />
                            </div>
                        {/each}
                    </div>
                {/if}

                <!-- DOCUMENTATION INJECTION -->
                <div>
                    <label class="label"><span class="label-text font-bold">Assign Edge Servers</span></label>
                    <p class="text-sm opacity-70 mb-4">Selected guides will be linked in the user's welcome email.</p>
                    <div class="space-y-2 max-h-64 overflow-y-auto">
                        {#each pages as page}
                            <label class="flex items-center gap-3 p-3 border border-base-200 rounded-lg cursor-pointer hover:bg-base-200/50 {selectedDocs.includes(page.Slug) ? 'bg-base-200' : ''}">
                                <input type="checkbox" class="checkbox checkbox-sm" checked={selectedDocs.includes(page.Slug)} on:change={() => toggleDoc(page.Slug)} />
                                <span class="font-mono text-sm">{page.Title}</span>
                            </label>
                        {:else}
                            <div class="text-sm opacity-50 p-4 border border-dashed rounded-lg">No CMS pages created yet.</div>
                        {/each}
                    </div>
                </div>
            </div>
            
            <div class="pt-4 flex justify-end">
                <button type="submit" class="btn btn-primary btn-lg px-12" disabled={isInviting}>
                    {#if isInviting} <span class="loading loading-spinner"></span> {/if}
                    Generate & Send Invite
                </button>
            </div>
        </form>
    </div>
</div>