<script>
    import { onMount } from 'svelte';
    
    let users = [];
    let isInviting = false;
    let alertMsg = '';
    
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchUsers() {
        const res = await fetch('/api/admin/users', { headers });
        if (res.ok) users = await res.json();
    }

    onMount(fetchUsers);

    async function handleInvite(e) {
        e.preventDefault();
        const form = e.target;
        const payload = {
            username: form.username.value,
            email: form.email.value,
            provision_gitea: form.provGitea.checked,
            edge_server_ids: form.servers.value.split(',').map(s => s.trim()).filter(Boolean)
        };

        try {
            const res = await fetch('/api/admin/users/invite', { method: 'POST', headers, body: JSON.stringify(payload) });
            if (!res.ok) throw new Error("Failed to invite user");
            alertMsg = "User invited successfully!";
            form.reset();
            fetchUsers();
        } catch (err) { alertMsg = err.message; }
    }
</script>

<div class="space-y-6">
    <div class="flex justify-between items-center">
        <div>
            <h1 class="text-3xl font-bold">Users & Access</h1>
            <p class="text-base-content/70 mt-1">Manage identities and their provisioned edge resources.</p>
        </div>
    </div>

    <!-- Provisioning Accordion -->
    <div class="collapse collapse-arrow bg-base-100 border border-base-300 shadow-sm">
        <input type="checkbox" /> 
        <div class="collapse-title text-lg font-medium">
            + Provision New User Access
        </div>
        <div class="collapse-content border-t border-base-200">
            <form on:submit={handleInvite} class="space-y-4 pt-4">
                {#if alertMsg}<div class="alert alert-success shadow-sm mb-4">{alertMsg}</div>{/if}
                
                <div class="grid grid-cols-2 gap-4">
                    <input type="text" name="username" required class="input input-bordered" placeholder="Username" />
                    <input type="email" name="email" required class="input input-bordered" placeholder="Email" />
                </div>
                
                <input type="text" name="servers" class="input input-bordered w-full font-mono" placeholder="Comma separated Agent UUIDs" />
                
                <div class="form-control bg-base-200/50 p-2 rounded-lg">
                    <label class="label cursor-pointer justify-start gap-4">
                        <input type="checkbox" name="provGitea" class="checkbox checkbox-primary checkbox-sm" checked />
                        <span class="label-text font-bold">Provision Gitea Account</span>
                    </label>
                </div>
                
                <button type="submit" class="btn btn-primary w-full">Generate Invite Link</button>
            </form>
        </div>
    </div>

    <!-- Identity Matrix Table -->
    <div class="card bg-base-100 shadow-sm border border-base-300">
        <div class="overflow-x-auto">
            <table class="table table-zebra w-full text-base-content text-lg">
                <thead class="bg-base-200">
                    <tr><th>Identity</th><th>Status</th><th>Gitea</th><th>Edge Agents</th></tr>
                </thead>
                <tbody>
                    {#each users as user}
                        <tr>
                            <td>
                                <div class="font-bold">{user.Username}</div>
                                <div class="text-sm opacity-50">{user.Email}</div>
                            </td>
                            <td><span class="badge {user.Status === 'ACTIVE' ? 'badge-success' : 'badge-warning'} badge-sm">{user.Status}</span></td>
                            <td>
                                {#if user.access_list.find(a => a.TargetType === 'GITEA')}
                                    <span class="badge badge-primary badge-sm">Provisioned</span>
                                {:else}
                                    <span class="text-xs text-gray-400">None</span>
                                {/if}
                            </td>
                            <td>
                                <div class="flex flex-wrap gap-1">
                                    {#each user.access_list.filter(a => a.TargetType === 'SERVER') as srv}
                                        <span class="badge badge-info badge-sm" title={srv.TargetID}>{srv.TargetID.substring(0, 8)}</span>
                                    {/each}
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    </div>
</div>