<script>
    import { createEventDispatcher } from 'svelte';
    
    export let macros = [];
    
    const dispatch = createEventDispatcher();
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    let activeUser = null;
    let isProcessing = false;

    // Form States
    let updateExpiryAmount = 0;
    let updateExpiryUnit = 'days';
    
    let selectedMacroId = '';
    let selectedMacroServer = '';
    
    let deprovPurgeRepos = false;
    let deprovPurgeHome = false;

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

    async function handleApplyMacro() {
        isProcessing = true;
        try {
            await fetch(`/api/admin/users/${activeUser.ID}/macro`, {
                method: 'POST', headers,
                body: JSON.stringify({ macro_id: selectedMacroId, server_id: selectedMacroServer })
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
        </div>
        <div class="modal-action">
            <button class="btn btn-ghost" on:click={() => close('macro')}>Cancel</button>
            <button class="btn btn-primary" on:click={handleApplyMacro} disabled={isProcessing || !selectedMacroId || !selectedMacroServer}>
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