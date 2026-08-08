<script>
    import { onMount } from 'svelte';
    
    let servers = [];
    let macros = [];
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    // Config Modal State
    let activeServer = null;
    let configProvId = '';
    let configSoftDeprovId = '';
    let configHardDeprovId = '';
    let isSaving = false;

    async function fetchData() {
        try {
            const [srvRes, macRes] = await Promise.all([
                fetch('/api/admin/servers', { headers }),
                fetch('/api/admin/macros', { headers })
            ]);
            if (srvRes.ok) servers = await srvRes.json();
            if (macRes.ok) macros = await macRes.json();
        } catch (err) { console.error("Fetch failed", err); }
    }

    onMount(fetchData);

    async function handleAddServer(e) {
        e.preventDefault();
        const form = e.target;
        try {
            await fetch('/api/admin/servers', {
                method: 'POST',
                headers,
                body: JSON.stringify({ name: form.serverName.value, public_key: form.publicKey.value })
            });
            form.reset();
            fetchData();
        } catch (err) { alert(err.message); }
    }

    async function handleDeregister(server) {
        if (!confirm(`Are you sure you want to deregister ${server.Name}? This does NOT deprovision users currently on it.`)) return;
        try {
            const res = await fetch(`/api/admin/servers/${server.ID}`, { method: 'DELETE', headers });
            if (!res.ok) throw new Error("Failed to deregister");
            fetchData();
        } catch (err) { alert(err.message); }
    }

    function openConfig(server) {
        activeServer = server;
        configProvId = server.ProvisionMacroID || '';
        configSoftDeprovId = server.SoftDeprovisionMacroID || '';
        configHardDeprovId = server.HardDeprovisionMacroID || '';
        document.getElementById('modal_agent_config').showModal();
    }

    async function saveConfig() {
        isSaving = true;
        try {
            const res = await fetch(`/api/admin/servers/${activeServer.ID}/config`, {
                method: 'PUT',
                headers,
                body: JSON.stringify({
                    provision_macro_id: configProvId,
                    soft_deprovision_macro_id: configSoftDeprovId,
                    hard_deprovision_macro_id: configHardDeprovId
                })
            });
            if (!res.ok) throw new Error("Failed to save config");
            document.getElementById('modal_agent_config').close();
            fetchData();
        } catch (err) { alert(err.message); }
        finally { isSaving = false; }
    }
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <div class="lg:col-span-2">
        <h1 class="text-3xl font-bold mb-1">Edge Agents</h1>
        <p class="text-base-content/70 mb-6">Manage infrastructure and bind lifecycle macros to agents.</p>
        
        <div class="card bg-base-100 shadow-sm border border-base-300 text-base-content">
            <table class="table w-full">
                <thead class="bg-base-200">
                    <tr><th>Status</th><th>Agent Identity</th><th>Bound Macros</th><th>Actions</th></tr>
                </thead>
                <tbody>
                    {#each servers as server}
                        <tr>
                            <td>
                                {#if server.Status === 'ONLINE'}
                                    <div class="badge badge-success gap-1"><span class="w-2 h-2 rounded-full bg-white"></span> Online</div>
                                {:else}
                                    <div class="badge badge-error">Offline</div>
                                {/if}
                            </td>
                            <td>
                                <div class="font-bold">{server.Name}</div>
                                <div class="font-mono text-xs opacity-50">
                                    {server.ID === 'internal-gitea' ? 'Virtual System' : server.ID.substring(0,8)}
                                </div>
                            </td>
                            <td>
                                <div class="flex flex-col gap-1 text-xs">
                                    {#if server.ProvisionMacroID}<span class="text-success font-mono">PROV: Configured</span>{/if}
                                    {#if server.SoftDeprovisionMacroID}<span class="text-warning font-mono">SOFT: Configured</span>{/if}
                                    {#if server.HardDeprovisionMacroID}<span class="text-error font-mono">HARD: Configured</span>{/if}
                                    {#if !server.ProvisionMacroID && !server.SoftDeprovisionMacroID && !server.HardDeprovisionMacroID}
                                        <span class="opacity-50">No macros bound</span>
                                    {/if}
                                </div>
                            </td>
                            <td>
                                <div class="flex gap-2">
                                    <button class="btn btn-sm btn-neutral" on:click={() => openConfig(server)}>Configure</button>
                                    {#if server.ID !== 'internal-gitea'}
                                        <button class="btn btn-sm btn-square btn-outline btn-error" on:click={() => handleDeregister(server)}>✕</button>
                                    {/if}
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    </div>

    <!-- Registration Sidebar -->
    <div>
        <div class="card bg-base-100 shadow-sm border border-base-300 mt-14 text-base-content">
            <div class="card-body bg-base-200/30 rounded-xl">
                <h3 class="font-bold text-lg mb-2">Register Agent</h3>
                <form on:submit={handleAddServer} class="space-y-4">
                    <input type="text" name="serverName" required class="input input-bordered input-md w-full" placeholder="e.g. prod-db-node" />
                    <textarea name="publicKey" required class="textarea textarea-bordered h-24 font-mono text-sm w-full" placeholder="Paste Ed25519 Public Key"></textarea>
                    <button type="submit" class="btn btn-neutral w-full">Authorize Agent</button>
                </form>
            </div>
        </div>
    </div>
</div>

<!-- Configuration Modal -->
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
                <label class="label">
                    <span class="label-text font-bold text-warning">Soft Deprovision Macro</span>
                </label>
                <select bind:value={configSoftDeprovId} class="select select-bordered w-full">
                    <option value="">None (Skip Soft Deprovision)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
                <label class="label"><span class="label-text-alt opacity-70">Used when "Destructive Purge" is UNCHECKED.</span></label>
            </div>

            <div class="form-control border-t border-base-300 pt-4 mt-2">
                <label class="label">
                    <span class="label-text font-bold text-error">Hard Purge Deprovision Macro</span>
                </label>
                <select bind:value={configHardDeprovId} class="select select-bordered w-full">
                    <option value="">None (Skip Hard Deprovision)</option>
                    {#each macros as m}<option value={m.ID}>{m.Name}</option>{/each}
                </select>
                <label class="label"><span class="label-text-alt opacity-70">Used when "Destructive Purge" is CHECKED.</span></label>
            </div>
        </div>

        <div class="modal-action">
            <button class="btn btn-ghost" on:click={() => document.getElementById('modal_agent_config').close()}>Cancel</button>
            <button class="btn btn-primary" on:click={saveConfig} disabled={isSaving}>
                {#if isSaving} <span class="loading loading-spinner loading-sm"></span> {/if}
                Save Configuration
            </button>
        </div>
    </div>
</dialog>