<script>
    import { createEventDispatcher } from 'svelte';
    export let macros = [];

    const dispatch = createEventDispatcher();
    const headers = { 
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + localStorage.getItem('nexus_jwt') 
    };

    let activeServer = null;
    let configName = '';
    let configAddress = ''; // NEW
    let configProvId = '';
    let configSoftDeprovId = '';
    let configHardDeprovId = '';
    let isSaving = false;

    export function open(server) {
        activeServer = server;
        configName = server.Name
        configAddress = server.Address || ''; // NEW
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
                    name: configName,
                    address: configAddress, // NEW
                    provision_macro_id: configProvId,
                    soft_deprovision_macro_id: configSoftDeprovId,
                    hard_deprovision_macro_id: configHardDeprovId
                })
            });
            if (!res.ok) throw new Error(`Failed to save config: ${await res.text()}`);
            close();
            dispatch('refresh');
        } catch (err) { alert(err.message); }
        finally { isSaving = false; }
    }
</script>

<dialog id="modal_agent_config" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Configure {activeServer?.Name}</h3>
        
        <div class="space-y-4">
            <div class="form-control mb-4">
                <label class="label"><span class="label-text font-bold pb-1">Server Name</span></label>
                <input type="text" bind:value={configName} required pattern="[a-zA-Z0-9_-]+" title="No spaces allowed (Use hyphens or underscores)" class="input input-bordered input-md w-full font-mono" placeholder="Name (e.g. prod-db)" />
            </div>

            <div class="form-control mb-4">
                <label class="label"><span class="label-text font-bold pb-1">Network Address (IP/Hostname)</span></label>
                <input type="text" bind:value={configAddress} class="input input-bordered w-full font-mono" placeholder="e.g. 192.168.1.50" />
                <label class="label"><span class="label-text-alt opacity-70">This value is injected into markdown documentation variables.</span></label>
            </div>

            <div class="divider">Pipeline Assignments</div>

            <div class="form-control">
                <label class="label"><span class="label-text font-bold text-success pb-1">Onboarding Macro</span></label>
                <select bind:value={configProvId} class="select select-bordered w-full">
                    <option value="">None (Skip Provisioning)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
            </div>

            <div class="form-control">
                <label class="label"><span class="label-text font-bold text-warning pb-1">Soft Deprovision Macro</span></label>
                <select bind:value={configSoftDeprovId} class="select select-bordered w-full">
                    <option value="">None (Skip Soft Deprovision)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
                <label class="label"><span class="label-text-alt opacity-70">Used when Destructive Purge is UNCHECKED.</span></label>
            </div>

            <div class="form-control border-t border-base-300 pt-4 mt-2">
                <label class="label"><span class="label-text font-bold text-error pb-1">Hard Purge Deprovision Macro</span></label>
                <select bind:value={configHardDeprovId} class="select select-bordered w-full">
                    <option value="">None (Skip Hard Deprovision)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
                <label class="label"><span class="label-text-alt opacity-70">Used when Destructive Purge is CHECKED.</span></label>
            </div>
        </div>

        <div class="modal-action mt-6">
            <button class="btn btn-ghost" on:click={close}>Cancel</button>
            <button class="btn btn-primary" on:click={saveConfig} disabled={isSaving}>
                {#if isSaving} <span class="loading loading-spinner loading-sm"></span> {/if} Save Configuration
            </button>
        </div>
    </div>
</dialog>