<script>
    import { onMount } from 'svelte';
    
    let servers = [];
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchServers() {
        const res = await fetch('/api/admin/servers', { headers });
        if (res.ok) servers = await res.json();
    }

    onMount(fetchServers);

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
            fetchServers();
        } catch (err) { alert(err.message); }
    }
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <div class="lg:col-span-2">
        <h1 class="text-3xl font-bold mb-1">Edge Agents</h1>
        <p class="text-base-content/70 mb-6">Connected edge resources and their declared lifecycle capabilities.</p>
        
        <div class="card bg-base-100 shadow-sm border border-base-300 text-base-content text-lg">
            <table class="table w-full">
                <thead class="bg-base-200">
                    <tr><th>Status</th><th>Agent Identity</th><th>Capabilities</th></tr>
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
                                <div class="font-mono text-xs opacity-50">{server.ID}</div>
                            </td>
                            <td>
                                <div class="flex flex-wrap gap-1 mt-1">
                                    {#if server.Capabilities && server.Capabilities !== "[]"}
                                        {#each JSON.parse(server.Capabilities) as cap}
                                            <span class="badge badge-outline badge-sm text-info">{cap}</span>
                                        {/each}
                                    {:else}
                                        <span class="text-xs opacity-50">No modules declared</span>
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
        <div class="card bg-base-100 shadow-sm border border-base-300 mt-14 text-base-content text-lg">
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