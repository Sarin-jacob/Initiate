<script>
    import { createEventDispatcher } from 'svelte';
    
    export let macros = [];
    
    const dispatch = createEventDispatcher();
    const headers = { 
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + localStorage.getItem('nexus_jwt') 
                };

    let activeUser = null;
    let isProcessing = false;

    // Form States
    let updateExpiryAmount = 0;
    let updateExpiryUnit = 'days';
    
    let selectedMacroId = '';
    let selectedMacroServer = '';
    
    let deprovPurgeRepos = false;
    let deprovPurgeHome = false;

    let adminInputs = {};
    let hasUserPrompts = false;
    let requiredAdminVars = [];

    // Exposed method for parent to open modals
    export function open(type, user) {
        activeUser = user;
        document.getElementById(`modal_${type}`).showModal();
    }

    function close(type) {
        document.getElementById(`modal_${type}`).close();
        activeUser = null;
    }

    async function handleExtendExpiry() {
        isProcessing = true;
        try {
            await fetch(`/api/admin/users/${activeUser.ID}/expire`, {
                method: 'PUT', headers,
                body: JSON.stringify({ expire_amount: parseInt(updateExpiryAmount) || 0, expire_unit: updateExpiryUnit })
            });
            close('expiry');
            dispatch('refresh');
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
    }

    $: if (selectedMacroId && macros.length > 0) {
        const macro = macros.find(m => m.ID === selectedMacroId);
        if (macro) {
            adminInputs = {};
            // Handle both Go capitalized 'Steps' or JSON lowercase 'steps'
            const stepsStr = typeof macro.Steps === 'string' ? macro.Steps : 
                             (typeof macro.steps === 'string' ? macro.steps : JSON.stringify(macro.Steps || macro.steps));
            
            // Check for user prompts
            hasUserPrompts = /\{\{user\.[a-zA-Z0-9_]+\}\}/.test(stepsStr);

            // Extract admin prompts
            const adminRegex = /\{\{admin\.([a-zA-Z0-9_]+)\}\}/g;
            let matches;
            const adminSet = new Set();
            while ((matches = adminRegex.exec(stepsStr)) !== null) {
                adminSet.add(matches[1]);
            }
            requiredAdminVars = Array.from(adminSet);
        }
    }

    async function handleApplyMacro() {
        isProcessing = true;
        try {
            await fetch(`/api/admin/users/${activeUser.ID}/macro`, {
                method: 'POST', headers,
                body: JSON.stringify({ 
                    macro_id: selectedMacroId, 
                    server_id: selectedMacroServer,
                    admin_inputs: adminInputs // NEW: Send the parsed inputs
                })
            });
            close('macro');
            dispatch('refresh');
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
    }

    async function handleDeprovision() {
        isProcessing = true;
        try {
            await fetch(`/api/admin/users/${activeUser.ID}/deprovision`, {
                method: 'POST', headers, 
                body: JSON.stringify({ purge_repos: deprovPurgeRepos, purge_home: deprovPurgeHome })
            });
            close('deprovision');
            dispatch('refresh');
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
    }
</script>

<!-- 1. Expiry Modal -->
<dialog id="modal_expiry" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Extend Expiration for {activeUser?.Username}</h3>
        <div class="form-control">
            <div class="join w-full">
                <input type="number" bind:value={updateExpiryAmount} min="0" placeholder="0 = Never" class="input input-bordered join-item w-full" />
                <select bind:value={updateExpiryUnit} class="select select-bordered join-item">
                    <option value="days">Days</option><option value="weeks">Weeks</option>
                    <option value="months">Months</option><option value="years">Years</option>
                </select>
            </div>
        </div>
        <div class="modal-action">
            <button class="btn btn-ghost" on:click={() => close('expiry')}>Cancel</button>
            <button class="btn btn-primary" on:click={handleExtendExpiry} disabled={isProcessing}>
                {#if isProcessing} <span class="loading loading-spinner loading-sm"></span> {/if} Update Date
            </button>
        </div>
    </div>
</dialog>

<!-- 2. Manual Macro Modal -->
<dialog id="modal_macro" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Run Manual Macro Pipeline</h3>
        <div class="space-y-4">
            <select bind:value={selectedMacroId} class="select select-bordered w-full">
                <option value="" disabled selected>1. Select Macro Pipeline</option>
                {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
            </select>
            
            <select bind:value={selectedMacroServer} class="select select-bordered w-full">
                <option value="" disabled selected>2. Select Target System</option>
                {#if activeUser?.access_list}
                    {#each activeUser.access_list as srv}
                        <option value={srv.TargetID}>
                            {srv.TargetType === 'GITEA' ? 'Central Gitea' : `Edge Agent: ${srv.TargetID.substring(0,8)}`}
                        </option>
                    {/each}
                {/if}
            </select>

            {#if requiredAdminVars.length > 0}
                <div class="divider text-sm">Administrator Inputs Required</div>
                {#each requiredAdminVars as adminVar}
                    <div class="form-control w-full">
                        <label class="label"><span class="label-text capitalize">{adminVar.replace(/_/g, ' ')}</span></label>
                        <input type="text" bind:value={adminInputs[adminVar]} class="input input-bordered w-full" required />
                    </div>
                {/each}
            {/if}

            {#if hasUserPrompts}
                <div class="alert alert-warning shadow-sm mt-4">
                    <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
                    <span class="text-sm">This macro requires end-user input (e.g., passwords). It cannot be run manually from the dashboard.</span>
                </div>
            {/if}
        </div>
        <div class="modal-action">
            <button class="btn btn-ghost" on:click={() => close('macro')}>Cancel</button>
            <button class="btn btn-primary" on:click={handleApplyMacro} 
                    disabled={isProcessing || !selectedMacroId || !selectedMacroServer || hasUserPrompts}>
                {#if isProcessing} <span class="loading loading-spinner loading-sm"></span> {/if} Execute Pipeline
            </button>
        </div>
    </div>
</dialog>

<!-- 3. Deprovision Modal -->
<dialog id="modal_deprovision" class="modal">
    <div class="modal-box border-t-4 border-error">
        <h3 class="font-bold text-lg text-error mb-2">Deprovision {activeUser?.Username}</h3>
        <div class="space-y-4 bg-base-200/50 p-4 rounded-xl border border-base-300">
            <p class="text-sm font-bold opacity-70 mb-2 uppercase">Destructive Purge Flags</p>
            <label class="cursor-pointer flex items-center gap-3">
                <input type="checkbox" bind:checked={deprovPurgeRepos} class="toggle toggle-error toggle-sm" />
                <span class="text-sm">Purge Git Repositories</span>
            </label>
            <label class="cursor-pointer flex items-center gap-3">
                <input type="checkbox" bind:checked={deprovPurgeHome} class="toggle toggle-error toggle-sm" />
                <span class="text-sm">Purge /home Directories</span>
            </label>
        </div>
        <div class="modal-action mt-6">
            <button class="btn btn-ghost" on:click={() => close('deprovision')}>Cancel</button>
            <button class="btn btn-error" on:click={handleDeprovision} disabled={isProcessing}>
                {#if isProcessing} <span class="loading loading-spinner loading-sm"></span> {/if} Confirm Deprovision
            </button>
        </div>
    </div>
</dialog>