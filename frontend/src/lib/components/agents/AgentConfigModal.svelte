<script>
    import { createEventDispatcher } from 'svelte';
    export let macros = [];
    
    const dispatch = createEventDispatcher();
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    let activeServer = null;
    let configProvId = '';
    let configSoftDeprovId = '';
    let configHardDeprovId = '';
    let isSaving = false;

    export function open(server) {
        activeServer = server;
        configProvId = server.ProvisionMacroID || '';
        configSoftDeprovId = server.SoftDeprovisionMacroID || '';
        configHardDeprovId = server.HardDeprovisionMacroID || '';
        document.getElementById('modal_agent_config').showModal();
    }

    function close() {
        activeServer = null;
        document.getElementById('modal_agent_config').close();
    }

    async function saveConfig() {
        isSaving = true;
        try {
            const res = await fetch(`/api/admin/servers/${activeServer.ID}/config`, {
                method: 'PUT', headers,
                body: JSON.stringify({
                    provision_macro_id: configProvId,
                    soft_deprovision_macro_id: configSoftDeprovId,
                    hard_deprovision_macro_id: configHardDeprovId
                })
            });
            if (!res.ok) throw new Error("Failed to save config");
            close();
            dispatch('refresh');
        } catch (err) { alert(err.message); }
        finally { isSaving = false; }
    }
</script>

<dialog id="modal_agent_config" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Configure {activeServer?.Name}</h3>
        <p class="text-sm opacity-70 mb-6">Assign the pipelines this agent should run when a user is granted or revoked access.</p>
        
        <div class="space-y-4">
            <div class="form-control">
                <label class="label"><span class="label-text font-bold text-success">Onboarding Macro</span></label>
                <select bind:value={configProvId} class="select select-bordered w-full">
                    <option value="">None (Skip Provisioning)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
            </div>
            
            <div class="form-control">
                <label class="label"><span class="label-text font-bold text-warning">Soft Deprovision Macro</span></label>
                <select bind:value={configSoftDeprovId} class="select select-bordered w-full">
                    <option value="">None (Skip Soft Deprovision)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
                <label class="label"><span class="label-text-alt opacity-70">Used when Destructive Purge is UNCHECKED.</span></label>
            </div>

            <div class="form-control border-t border-base-300 pt-4 mt-2">
                <label class="label"><span class="label-text font-bold text-error">Hard Purge Deprovision Macro</span></label>
                <select bind:value={configHardDeprovId} class="select select-bordered w-full">
                    <option value="">None (Skip Hard Deprovision)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
                <label class="label"><span class="label-text-alt opacity-70">Used when Destructive Purge is CHECKED.</span></label>
            </div>
        </div>

        <div class="modal-action">
            <button class="btn btn-ghost" on:click={close}>Cancel</button>
            <button class="btn btn-primary" on:click={saveConfig} disabled={isSaving}>
                {#if isSaving} <span class="loading loading-spinner loading-sm"></span> {/if} Save Configuration
            </button>
        </div>
    </div>
</dialog>