<script>
    import { onMount } from 'svelte';
    
    let users = [];
    let servers = [];
    
    // Tracks selected servers and their granted modules. 
    // Format: { "server-uuid": ["system_user", "ssh_key"] }
    let agentAllocations = {}; 
    
    let isInviting = false;
    let alertMsg = '';
    
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchData() {
        try {
            const [resUsers, resServers] = await Promise.all([
                fetch('/api/admin/users', { headers }),
                fetch('/api/admin/servers', { headers })
            ]);
            
            if (resUsers.ok) users = await resUsers.json() || [];
            if (resServers.ok) servers = await resServers.json() || [];
        } catch (err) {
            console.error("Failed to load data", err);
        }
    }

    onMount(fetchData);

    // Toggle server selection
    function toggleServer(serverId, capabilitiesStr) {
        if (agentAllocations[serverId]) {
            // Deselect server
            delete agentAllocations[serverId];
            agentAllocations = { ...agentAllocations };
        } else {
            // Select server and default to granting ALL its capabilities
            let caps = [];
            try { caps = JSON.parse(capabilitiesStr || "[]"); } catch (e) {}
            agentAllocations[serverId] = caps;
            agentAllocations = { ...agentAllocations };
        }
    }

    // Toggle specific module for a server
    function toggleModule(serverId, moduleName) {
        let current = agentAllocations[serverId] || [];
        if (current.includes(moduleName)) {
            agentAllocations[serverId] = current.filter(m => m !== moduleName);
        } else {
            agentAllocations[serverId] = [...current, moduleName];
        }
        agentAllocations = { ...agentAllocations };
    }

    async function handleInvite(e) {
        e.preventDefault();
        isInviting = true;
        const form = e.target;
        
        // Construct the new dynamic payload
        const payload = {
            username: form.username.value,
            email: form.email.value,
            provision_gitea: form.provGitea.checked,
            // Convert our dictionary into an array of objects for the backend
            edge_allocations: Object.keys(agentAllocations).map(serverId => ({
                server_id: serverId,
                modules: agentAllocations[serverId]
            }))
        };

        try {
            const res = await fetch('/api/admin/users/invite', { method: 'POST', headers, body: JSON.stringify(payload) });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to invite user");
            
            alertMsg = "User invited successfully!";
            form.reset();
            agentAllocations = {}; // Reset allocations
            fetchData();
        } catch (err) { 
            alertMsg = err.message; 
        } finally {
            isInviting = false;
        }
    }
</script>

<div class="space-y-8">
    <div class="flex justify-between items-center">
        <div>
            <h1 class="text-4xl font-bold">Users & Access</h1>
            <p class="text-base-content/70 mt-2 text-lg">Manage identities and their provisioned edge resources.</p>
        </div>
    </div>

    <!-- Provisioning Accordion -->
    <div class="collapse collapse-arrow bg-base-100 border border-base-300 shadow-sm">
        <input type="checkbox" /> 
        <div class="collapse-title text-xl font-bold p-6">
            + Provision New User Access
        </div>
        <div class="collapse-content border-t border-base-200 p-6">
            <form on:submit={handleInvite} class="space-y-6 pt-4">
                {#if alertMsg}<div class="alert alert-success shadow-sm mb-4">{alertMsg}</div>{/if}
                
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Username</span></label>
                        <input type="text" name="username" required class="input input-bordered input-lg" placeholder="jdoe" />
                    </div>
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Email Address</span></label>
                        <input type="email" name="email" required class="input input-bordered input-lg" placeholder="jdoe@company.com" />
                    </div>
                </div>
                
                <div class="form-control bg-base-200/50 p-4 rounded-xl border border-base-300">
                    <label class="label cursor-pointer justify-start gap-4">
                        <input type="checkbox" name="provGitea" class="checkbox checkbox-primary checkbox-lg" checked />
                        <span class="label-text font-bold text-lg">Provision Central Gitea Account</span>
                    </label>
                </div>

                <!-- Dynamic Agent Selector -->
                <div class="mt-6">
                    <h3 class="font-bold text-lg mb-4">Target Edge Agents & Capabilities</h3>
                    
                    {#if servers.length === 0}
                        <div class="p-6 bg-base-200 text-center rounded-xl opacity-70">
                            No Edge Agents registered. Go to the Agents tab to add one.
                        </div>
                    {:else}
                        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                            {#each servers as server}
                                <div class="card bg-base-100 border {agentAllocations[server.ID] ? 'border-primary ring-1 ring-primary' : 'border-base-300'} transition-all">
                                    <div class="card-body p-4">
                                        <!-- Server Toggle -->
                                        <label class="cursor-pointer flex items-center gap-4">
                                            <input type="checkbox" class="checkbox checkbox-primary" 
                                                checked={!!agentAllocations[server.ID]} 
                                                on:change={() => toggleServer(server.ID, server.Capabilities)} />
                                            <div class="flex-1">
                                                <div class="font-bold text-lg">{server.Name}</div>
                                                <div class="text-xs font-mono opacity-50">{server.ID.split('-')[0]}</div>
                                            </div>
                                            <span class="badge {server.Status === 'ONLINE' ? 'badge-success' : 'badge-error'} badge-sm">
                                                {server.Status}
                                            </span>
                                        </label>

                                        <!-- Granular Module Toggles (Revealed when server is checked) -->
                                        {#if agentAllocations[server.ID]}
                                            <div class="mt-4 pl-10 border-l-2 border-base-200 space-y-3">
                                                <div class="text-sm font-semibold opacity-70 mb-2">Granted Modules:</div>
                                                
                                                {#each JSON.parse(server.Capabilities || "[]") as cap}
                                                    <label class="cursor-pointer flex items-center gap-3">
                                                        <input type="checkbox" class="toggle toggle-sm toggle-success" 
                                                            checked={agentAllocations[server.ID].includes(cap)}
                                                            on:change={() => toggleModule(server.ID, cap)} />
                                                        <span class="font-mono text-sm">{cap}</span>
                                                    </label>
                                                {:else}
                                                    <div class="text-sm text-warning">No capabilities reported by agent.</div>
                                                {/each}
                                            </div>
                                        {/if}
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
                
                <div class="pt-4 flex justify-end">
                    <button type="submit" class="btn btn-primary btn-lg px-12" disabled={isInviting}>
                        {#if isInviting} <span class="loading loading-spinner"></span> {/if}
                        Generate Invite Link
                    </button>
                </div>
            </form>
        </div>
    </div>

    <!-- Identity Matrix Table -->
    <div class="card bg-base-100 shadow-sm border border-base-300">
        <div class="overflow-x-auto">
            <table class="table table-zebra w-full text-base">
                <thead class="bg-base-200 text-base">
                    <tr><th>Identity</th><th>Status</th><th>Gitea</th><th>Edge Agents</th></tr>
                </thead>
                <tbody>
                    {#each users as user}
                        <!-- Table content remains the same... -->
                        <tr>
                            <td>
                                <div class="font-bold text-lg">{user.Username}</div>
                                <div class="text-sm opacity-60">{user.Email}</div>
                            </td>
                            <td><span class="badge {user.Status === 'ACTIVE' ? 'badge-success' : 'badge-warning'} p-3">{user.Status}</span></td>
                            <td>
                                {#if user.access_list && user.access_list.find(a => a.TargetType === 'GITEA')}
                                    <span class="badge badge-primary p-3">Provisioned</span>
                                {:else}
                                    <span class="text-sm text-gray-400">None</span>
                                {/if}
                            </td>
                            <td>
                                <div class="flex flex-wrap gap-2">
                                    {#if user.access_list}
                                        {#each user.access_list.filter(a => a.TargetType === 'SERVER') as srv}
                                            <span class="badge badge-info p-3" title={srv.TargetID}>{srv.TargetID.substring(0, 8)}</span>
                                        {/each}
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