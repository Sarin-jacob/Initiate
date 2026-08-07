<script>
    import { onMount } from 'svelte';
    import Settings from './Settings.svelte';

    // --- State ---
    let activeTab = 'users'; // 'users', 'agents', 'provision'
    let users = [];
    let servers = [];
    
    // Provisioning Form State
    let isLoading = false;
    let alertMsg = '';
    let isError = false;
    let defaultMarkdown = `## Welcome **{{.Username}}**!
Please set your password below to finalize your system access. Your Gitea account will be registered under \`{{.Email}}\`.`;

    // Agent Registration State
    let isAddingServer = false;

    // Authentication Header (Matches your RequireAdmin middleware logic)
    const headers = {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-admin'
    };

    // --- Data Fetching ---
    async function fetchData() {
        try {
            const [resUsers, resServers] = await Promise.all([
                fetch('/api/admin/users', { headers }),
                fetch('/api/admin/servers', { headers })
            ]);
            
            if (resUsers.ok) users = await resUsers.json();
            if (resServers.ok) servers = await resServers.json();
        } catch (err) {
            console.error("Failed to load dashboard data", err);
        }
    }

    onMount(fetchData);

    // --- Handlers ---
    async function handleInvite(e) {
        e.preventDefault();
        isLoading = true;
        alertMsg = '';

        const form = e.target;
        const payload = {
            username: form.username.value,
            email: form.email.value,
            provision_gitea: form.provGitea.checked,
            edge_server_ids: form.servers.value.split(',').map(s => s.trim()).filter(Boolean),
            markdown_template: defaultMarkdown
        };

        try {
            const res = await fetch('/api/admin/users/invite', {
                method: 'POST',
                headers,
                body: JSON.stringify(payload)
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || res.statusText);
            
            alertMsg = 'Success! User invited.';
            isError = false;
            form.reset();
            await fetchData(); // Refresh the users table
        } catch (err) {
            alertMsg = err.message;
            isError = true;
        } finally {
            isLoading = false;
        }
    }

    async function handleAddServer(e) {
        e.preventDefault();
        isAddingServer = true;
        const form = e.target;

        try {
            const res = await fetch('/api/admin/servers', {
                method: 'POST',
                headers,
                body: JSON.stringify({
                    name: form.serverName.value,
                    public_key: form.publicKey.value
                })
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to add server");
            
            form.reset();
            await fetchData(); // Refresh the agents table
        } catch (err) {
            alert(err.message);
        } finally {
            isAddingServer = false;
        }
    }
</script>

<div class="container mx-auto p-6 max-w-6xl">
    <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold">NexusIAM Admin</h1>
        <div class="badge badge-success badge-outline gap-2">
            <div class="w-2 h-2 rounded-full bg-success"></div>
            System Online
        </div>
    </div>

    <!-- Tabs Navigation -->
    <div role="tablist" class="tabs tabs-lifted mb-6">
        <button role="tab" class="tab text-lg {activeTab === 'users' ? 'tab-active font-bold' : ''}" on:click={() => activeTab = 'users'}>User Access Matrix</button>
        <button role="tab" class="tab text-lg {activeTab === 'agents' ? 'tab-active font-bold' : ''}" on:click={() => activeTab = 'agents'}>Edge Agents</button>
        <button role="tab" class="tab text-lg {activeTab === 'provision' ? 'tab-active font-bold' : ''}" on:click={() => activeTab = 'provision'}>Provision New User</button>
        <button role="tab" class="tab text-lg {activeTab === 'settings' ? 'tab-active font-bold' : ''}" on:click={() => activeTab = 'settings'}>Templates / CMS</button>
    </div>

    <!-- TAB 1: USERS MATRIX -->
    {#if activeTab === 'users'}
        <div class="card bg-base-100 shadow-xl border border-base-300">
            <div class="card-body">
                <h2 class="card-title mb-4">Identity & Access Matrix</h2>
                <div class="overflow-x-auto">
                    <table class="table table-zebra w-full">
                        <thead>
                            <tr class="bg-base-200">
                                <th>User</th>
                                <th>Account Status</th>
                                <th>Gitea Provisioning</th>
                                <th>Edge Servers</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each users as user}
                                <tr>
                                    <td>
                                        <div class="flex items-center gap-3">
                                            <div class="avatar placeholder">
                                              <div class="bg-neutral text-neutral-content rounded-full w-10">
                                                <span>{user.Username.charAt(0).toUpperCase()}</span>
                                              </div>
                                            </div>
                                            <div>
                                                <div class="font-bold">{user.Username}</div>
                                                <div class="text-sm opacity-50">{user.Email}</div>
                                            </div>
                                        </div>
                                    </td>
                                    <td>
                                        <span class="badge {user.Status === 'ACTIVE' ? 'badge-success' : 'badge-warning'} badge-sm">
                                            {user.Status}
                                        </span>
                                    </td>
                                    <td>
                                        <!-- Look for Gitea access in their access list -->
                                        {#if user.access_list.find(a => a.TargetType === 'GITEA')}
                                            {#if user.access_list.find(a => a.TargetType === 'GITEA').Status === 'ACTIVE'}
                                                <span class="badge badge-primary badge-sm">Provisioned</span>
                                            {:else}
                                                <span class="badge badge-ghost badge-sm">Pending</span>
                                            {/if}
                                        {:else}
                                            <span class="text-xs text-gray-400">None</span>
                                        {/if}
                                    </td>
                                    <td>
                                        <div class="flex flex-wrap gap-1">
                                            {#each user.access_list.filter(a => a.TargetType === 'SERVER') as srv}
                                                <span class="badge {srv.Status === 'ACTIVE' ? 'badge-info' : 'badge-ghost'} badge-sm" title={srv.TargetID}>
                                                    {srv.TargetID.substring(0, 8)}... ({srv.Status})
                                                </span>
                                            {/each}
                                            {#if user.access_list.filter(a => a.TargetType === 'SERVER').length === 0}
                                                <span class="text-xs text-gray-400">No servers</span>
                                            {/if}
                                        </div>
                                    </td>
                                </tr>
                            {:else}
                                <tr><td colspan="4" class="text-center py-4 text-gray-500">No users found.</td></tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    {/if}

    <!-- TAB 2: EDGE AGENTS -->
    {#if activeTab === 'agents'}
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div class="lg:col-span-2 card bg-base-100 shadow-xl border border-base-300">
                <div class="card-body">
                    <h2 class="card-title mb-4">Registered Target Servers</h2>
                    <div class="overflow-x-auto">
                        <table class="table w-full">
                            <thead>
                                <tr class="bg-base-200">
                                    <th>Status</th>
                                    <th>Name</th>
                                    <th>ID (Target)</th>
                                    <th>Public Key (Truncated)</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each servers as server}
                                    <tr>
                                        <td>
                                            {#if server.Status === 'ONLINE'}
                                                <div class="badge badge-success gap-1"><span class="w-2 h-2 rounded-full bg-white"></span> Online</div>
                                            {:else}
                                                <div class="badge badge-error gap-1">Offline</div>
                                            {/if}
                                        </td>
                                        <td>
                                            <div class="font-bold">{server.Name}</div>
                                            <!-- Parse the JSON string into tags -->
                                            <div class="flex gap-1 mt-1">
                                                {#if server.Capabilities && server.Capabilities !== "[]"}
                                                    {#each JSON.parse(server.Capabilities || "[]") as cap}
                                                        <span class="badge badge-outline badge-xs text-info">{cap}</span>
                                                    {/each}
                                                {:else}
                                                    <span class="text-xs text-gray-400">No capabilities reported</span>
                                                {/if}
                                            </div>
                                        </td>
                                        <td class="font-mono text-xs">{server.ID}</td>
                                        <td class="font-mono text-xs text-gray-500" title={server.PublicKey}>
                                            {server.PublicKey.substring(0, 16)}...
                                        </td>
                                    </tr>
                                {:else}
                                    <tr><td colspan="4" class="text-center py-4 text-gray-500">No edge agents registered.</td></tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <!-- Add Server Form -->
            <div class="card bg-base-100 shadow-xl border border-base-300 h-fit">
                <div class="card-body bg-base-200/50 rounded-xl">
                    <h3 class="font-bold text-lg mb-2">Register New Agent</h3>
                    <form on:submit={handleAddServer} class="space-y-4">
                        <div class="form-control">
                            <label class="label"><span class="label-text">Agent Name</span></label>
                            <input type="text" name="serverName" required class="input input-bordered input-sm" placeholder="e.g. prod-db-node" />
                        </div>
                        <div class="form-control">
                            <label class="label"><span class="label-text">Ed25519 Public Key (Hex)</span></label>
                            <textarea name="publicKey" required class="textarea textarea-bordered h-24 font-mono text-xs" placeholder="Paste hex output from keygen..."></textarea>
                        </div>
                        <button type="submit" class="btn btn-neutral w-full" disabled={isAddingServer}>
                            {#if isAddingServer} <span class="loading loading-spinner loading-sm"></span> {/if}
                            Add to Network
                        </button>
                    </form>
                </div>
            </div>
        </div>
    {/if}

    <!-- TAB 3: PROVISIONING -->
    {#if activeTab === 'provision'}
        <div class="card bg-base-100 shadow-xl border border-base-300 max-w-4xl mx-auto">
            <div class="card-body">
                <h2 class="card-title text-2xl mb-4">Provision New User Access</h2>
                
                {#if alertMsg}
                    <div class="alert {isError ? 'alert-error' : 'alert-success'} shadow-sm mb-6">
                        <span>{alertMsg}</span>
                    </div>
                {/if}

                <form on:submit={handleInvite} class="space-y-4">
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div class="form-control">
                            <label class="label"><span class="label-text font-semibold">Username</span></label>
                            <input type="text" name="username" required class="input input-bordered w-full" placeholder="jdoe" />
                        </div>
                        <div class="form-control">
                            <label class="label"><span class="label-text font-semibold">Email</span></label>
                            <input type="email" name="email" required class="input input-bordered w-full" placeholder="user@company.com" />
                        </div>
                    </div>

                    <div class="form-control">
                        <label class="label"><span class="label-text font-semibold">Target Edge Server IDs</span></label>
                        <input type="text" name="servers" class="input input-bordered w-full font-mono" placeholder="agent-uuid-1, agent-uuid-2" />
                        <label class="label"><span class="label-text-alt text-gray-500">Comma separated IDs. Check the Edge Agents tab for valid UUIDs.</span></label>
                    </div>

                    <div class="form-control bg-base-200 p-4 rounded-lg">
                        <label class="label cursor-pointer justify-start gap-4">
                            <input type="checkbox" name="provGitea" class="checkbox checkbox-primary" checked />
                            <span class="label-text font-bold">Provision Central Gitea Account</span>
                        </label>
                    </div>

                    <div class="form-control">
                        <label class="label"><span class="label-text font-semibold">Onboarding Guide (Markdown)</span></label>
                        <textarea name="markdown" bind:value={defaultMarkdown} class="textarea textarea-bordered font-mono h-32" required></textarea>
                    </div>

                    <div class="card-actions justify-end mt-6">
                        <button type="submit" class="btn btn-primary w-full md:w-auto" disabled={isLoading}>
                            {#if isLoading} <span class="loading loading-spinner"></span> {/if}
                            Send Invitation
                        </button>
                    </div>
                </form>
            </div>
        </div>
    {/if}
    {#if activeTab === 'settings'}
        <Settings />
    {/if}
</div>