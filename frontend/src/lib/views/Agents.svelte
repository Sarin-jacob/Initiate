<script>
    import { onMount } from 'svelte';
    import AgentConfigModal from '../components/agents/AgentConfigModal.svelte';

    let servers = [];
    let macros = [];
    let configModalRef;

    const getHeaders = () => ({ 
            'Content-Type': 'application/json', 
            'Authorization': 'Bearer ' + localStorage.getItem('nexus_jwt') 
        });

    async function fetchData() {
        try {
            const [srvRes, macRes] = await Promise.all([
                fetch('/api/admin/servers', { headers: getHeaders() }),
                fetch('/api/admin/macros', { headers: getHeaders() })
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
            const res = await fetch('/api/admin/servers', {
                method: 'POST', headers: getHeaders(),
                body: JSON.stringify({ 
                    name: form.serverName.value, 
                    address: form.serverAddress.value,
                    public_key: form.publicKey.value 
                })
            });
            if (!res.ok) throw new Error(await res.text());
            form.reset();
            fetchData();
        } catch (err) { alert(err.message); }
    }

    async function handleDeregister(server) {
        if (!confirm(`Deregister ${server.Name}? This does NOT deprovision users currently on it.`)) return;
        try {
            await fetch(`/api/admin/servers/${server.ID}`, { method: 'DELETE', headers: getHeaders() });
            fetchData();
        } catch (err) { alert(err.message); }
    }
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <div class="lg:col-span-2">
        <h1 class="text-3xl font-bold mb-1">Edge Agents</h1>
        <p class="text-base-content/70 mb-6">Manage infrastructure and bind lifecycle macros to agents.</p>

        <div class="card bg-base-100 shadow-sm border border-base-300 text-base-content">
            <div class="overflow-x-auto">
                <table class="table w-full">
                    <thead class="bg-base-200">
                        <tr>
                            <th>Status</th>
                            <th>Agent Identity</th>
                            <th>Network Address</th>
                            <th>Bound Macros</th>
                            <th class="text-right">Actions</th>
                        </tr>
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
                                    {#if server.Address}
                                        <span class="badge badge-ghost font-mono text-xs p-3 border-base-300 shadow-sm">{server.Address}</span>
                                    {:else}
                                        <span class="text-xs italic opacity-50 text-warning">Unconfigured</span>
                                    {/if}
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
                                <td class="text-right">
                                    <div class="flex gap-2 justify-end">
                                        <button class="btn btn-sm btn-neutral" on:click={() => configModalRef.open(server)}>Configure</button>
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
    </div>

    <div>
        <div class="card bg-base-100 shadow-sm border border-base-300 mt-14 text-base-content">
            <div class="card-body bg-base-200/30 rounded-xl">
                <h3 class="font-bold text-lg mb-2">Register Agent</h3>
                <form on:submit={handleAddServer} class="space-y-4">
                    <div class="grid grid-cols-2 gap-4">
                        <input type="text" name="serverName" required pattern="[a-zA-Z0-9_-]+" title="No spaces allowed (Use hyphens or underscores)" class="input input-bordered input-md w-full font-mono" placeholder="Name (e.g. prod-db)" />
                        <input type="text" name="serverAddress" required class="input input-bordered input-md w-full font-mono" placeholder="IP (e.g. 10.0.0.5)" />
                    </div>
                    <textarea name="publicKey" required class="textarea textarea-bordered h-24 font-mono text-sm w-full" placeholder="Paste Ed25519 Public Key"></textarea>
                    <button type="submit" class="btn btn-neutral w-full">Authorize Agent</button>
                </form>
            </div>
        </div>
    </div>
</div>

<AgentConfigModal bind:this={configModalRef} {macros} on:refresh={fetchData} />