<script>
    import { onMount } from 'svelte';
    
    let users = [];
    let servers = [];
    let macros = [];
    
    // Tracks selected servers and their granted modules. 
    // Format: { "server-uuid": ["system_user", "ssh_key"] }
    let agentAllocations = {}; 
    
    // Action Modal State
    let activeUser = null; 
    let isProcessing = false;

    // Deprovision Form State
    let deprovGitea = true;
    let deprovPurgeRepos = false;
    let deprovPurgeHome = false;

    // Macro Form State
    let selectedMacroId = '';
    let selectedMacroServer = '';

    // Expiry Form State
    let updateExpiryAmount = 0;
    let updateExpiryUnit = 'days';

    let isInviting = false;
    let alertMsg = '';
    
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchData() {
        try {
            const [resUsers, resServers, resMacros] = await Promise.all([
                fetch('/api/admin/users', { headers }),
                fetch('/api/admin/servers', { headers }),
                fetch('/api/admin/macros', { headers })
            ]);
            
            if (resUsers.ok) users = await resUsers.json() || [];
            if (resServers.ok) servers = await resServers.json() || [];
            if (resMacros.ok) macros = await resMacros.json() || [];
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
        
        const payload = {
            username: form.username.value,
            email: form.email.value,
            provision_gitea: form.provGitea.checked,
            expire_amount: parseInt(form.expireAmount.value) || 0,
            expire_unit: form.expireUnit.value,
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
            agentAllocations = {}; 
            fetchData();
        } catch (err) { alertMsg = err.message; }
        finally { isInviting = false; }
    }

    // Format Expiration Date for the table
    function formatExpiry(dateStr) {
        if (!dateStr) return "Never";
        const d = new Date(dateStr);
        if (d < new Date()) return "Expired";
        return d.toLocaleDateString();
    }

    // --- MODAL CONTROLS ---

    function openModal(modalId, user) {
        activeUser = user;
        document.getElementById(modalId).showModal();
    }

    function closeModal(modalId) {
        activeUser = null;
        document.getElementById(modalId).close();
    }

    // --- ACTION HANDLERS ---

    async function handleExtendExpiry() {
        isProcessing = true;
        try {
            const res = await fetch(`/api/admin/users/${activeUser.ID}/expire`, {
                method: 'PUT',
                headers,
                body: JSON.stringify({
                    expire_amount: parseInt(updateExpiryAmount) || 0,
                    expire_unit: updateExpiryUnit
                })
            });
            if (!res.ok) throw new Error("Failed to update expiration");
            closeModal('modal_expiry');
            fetchData();
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
    }

    async function handleApplyMacro() {
        if (!selectedMacroId || !selectedMacroServer) return alert("Select macro and server.");
        isProcessing = true;
        try {
            const res = await fetch(`/api/admin/users/${activeUser.ID}/macro`, {
                method: 'POST',
                headers,
                body: JSON.stringify({ macro_id: selectedMacroId, server_id: selectedMacroServer })
            });
            if (!res.ok) throw new Error("Failed to execute macro pipeline");
            closeModal('modal_macro');
            fetchData();
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
    }

    async function handleDeprovision() {
        if (!confirm(`WARNING: Are you sure you want to deprovision ${activeUser.Username}?`)) return;
        
        isProcessing = true;
        // Construct the teardown payload based on the user's current server access
        const serversToTeardown = (activeUser.access_list || [])
            .filter(a => a.TargetType === 'SERVER')
            .map(a => ({ target_id: a.TargetID, purge_home: deprovPurgeHome }));

        const payload = {
            gitea: { enabled: deprovGitea, purge_repos: deprovPurgeRepos },
            servers: serversToTeardown
        };

        try {
            const res = await fetch(`/api/admin/users/${activeUser.ID}/deprovision`, {
                method: 'POST',
                headers,
                body: JSON.stringify(payload)
            });
            if (!res.ok) throw new Error("Deprovisioning failed");
            closeModal('modal_deprovision');
            fetchData();
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
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
        <div class="collapse-title text-xl font-bold p-6">+ Provision New User Access</div>
        <div class="collapse-content border-t border-base-200 p-6">
            <form on:submit={handleInvite} class="space-y-6 pt-4">
                {#if alertMsg}<div class="alert alert-success shadow-sm mb-4">{alertMsg}</div>{/if}
                
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Username</span></label>
                        <input type="text" name="username" required class="input input-bordered input-lg" />
                    </div>
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Email Address</span></label>
                        <input type="email" name="email" required class="input input-bordered input-lg" />
                    </div>
                </div>
                
                <!-- Expiration Controls -->
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div class="form-control bg-base-200/50 p-4 rounded-xl border border-base-300">
                        <label class="label cursor-pointer justify-start gap-4">
                            <input type="checkbox" name="provGitea" class="checkbox checkbox-primary checkbox-lg" checked />
                            <span class="label-text font-bold text-lg">Provision Central Gitea Account</span>
                        </label>
                    </div>

                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Automated Expiration</span></label>
                        <div class="join w-full h-fit">
                            <input type="number" name="expireAmount" min="0" placeholder="0 = Never" class="input input-bordered join-item w-full input-lg" />
                            <select name="expireUnit" class="select select-bordered join-item input-lg h-auto">
                                <option value="days">Days</option>
                                <option value="weeks">Weeks</option>
                                <option value="months">Months</option>
                                <option value="years">Years</option>
                            </select>
                        </div>
                    </div>
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
                    <tr><th>Identity</th><th>Status</th><th>Gitea</th><th>Expires</th><th>Edge Agents</th><th>Actions</th></tr>
                </thead>
                <tbody>
                    {#each users as user}
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
                            
                            <!-- NEW: Expiration Display -->
                            <td>
                                <span class="text-sm font-mono {formatExpiry(user.ExpiresAt) === 'Expired' ? 'text-error font-bold' : ''}">
                                    {formatExpiry(user.ExpiresAt)}
                                </span>
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
                            
                            <!-- NEW: User Actions Dropdown -->
                            <td class="w-16">
                                <div class="dropdown dropdown-end">
                                    <div tabindex="0" role="button" class="btn btn-ghost btn-sm btn-circle">
                                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 12.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 18.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z" /></svg>
                                    </div>
                                    <ul class="dropdown-content z-[1] menu p-2 shadow-lg bg-base-100 rounded-box w-56 border border-base-300">
                                        <li class="menu-title px-4 py-2">Manage {user.Username}</li>
                                        <li><button type="button" on:click|preventDefault={() => openModal('modal_expiry', user)}>Extend Expiration</button></li>
                                        <li><button type="button" on:click|preventDefault={() => openModal('modal_macro', user)}>Apply New Macro</button></li>
                                        <div class="divider my-1"></div>
                                        <li><button type="button" class="text-error font-bold" on:click|preventDefault={() => openModal('modal_deprovision', user)}>Deprovision User</button></li>
                                    </ul>
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    </div>
</div>

<!-- 1. Expiration Modal -->
<dialog id="modal_expiry" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Extend Expiration for {activeUser?.Username}</h3>
        <div class="form-control">
            <div class="join w-full">
                <input type="number" bind:value={updateExpiryAmount} min="0" placeholder="0 = Never" class="input input-bordered join-item w-full" />
                <select bind:value={updateExpiryUnit} class="select select-bordered join-item">
                    <option value="days">Days</option>
                    <option value="weeks">Weeks</option>
                    <option value="months">Months</option>
                    <option value="years">Years</option>
                </select>
            </div>
            <label class="label"><span class="label-text-alt opacity-70">Set to 0 to remove expiration completely.</span></label>
        </div>
        <div class="modal-action">
            <button class="btn btn-ghost" on:click={() => closeModal('modal_expiry')}>Cancel</button>
            <button class="btn btn-primary" on:click={handleExtendExpiry} disabled={isProcessing}>
                {#if isProcessing} <span class="loading loading-spinner loading-sm"></span> {/if}
                Update Date
            </button>
        </div>
    </div>
</dialog>

<!-- 2. Apply Macro Modal -->
<dialog id="modal_macro" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Run Macro Pipeline</h3>
        <p class="text-sm opacity-70 mb-4">Execute a sequence of actions on a specific edge agent for <b>{activeUser?.Username}</b>.</p>
        
        <div class="space-y-4">
            <select bind:value={selectedMacroId} class="select select-bordered w-full">
                <option value="" disabled selected>1. Select Macro Pipeline</option>
                {#each macros as m}
                    <option value={m.ID}>{m.Name}</option>
                {/each}
            </select>
            
            <select bind:value={selectedMacroServer} class="select select-bordered w-full">
                <option value="" disabled selected>2. Select Target Edge Agent</option>
                {#if activeUser?.access_list}
                    {#each activeUser.access_list.filter(a => a.TargetType === 'SERVER') as srv}
                        <option value={srv.TargetID}>Agent UUID: {srv.TargetID}</option>
                    {/each}
                {/if}
            </select>
        </div>
        
        <div class="modal-action">
            <button class="btn btn-ghost" on:click={() => closeModal('modal_macro')}>Cancel</button>
            <button class="btn btn-primary" on:click={handleApplyMacro} disabled={isProcessing || !selectedMacroId || !selectedMacroServer}>
                {#if isProcessing} <span class="loading loading-spinner loading-sm"></span> {/if}
                Execute Pipeline
            </button>
        </div>
    </div>
</dialog>

<!-- 3. Deprovision Modal -->
<dialog id="modal_deprovision" class="modal">
    <div class="modal-box border-t-4 border-error">
        <h3 class="font-bold text-lg text-error mb-2">Deprovision {activeUser?.Username}</h3>
        <p class="text-sm opacity-70 mb-6">This will immediately revoke access. You can choose to preserve or purge their digital footprint.</p>
        
        <div class="space-y-4 bg-base-200/50 p-4 rounded-xl border border-base-300">
            <label class="cursor-pointer flex items-center gap-4">
                <input type="checkbox" bind:checked={deprovGitea} class="checkbox checkbox-error" />
                <span class="font-bold flex-1">Revoke Gitea Access</span>
            </label>
            {#if deprovGitea}
                <div class="pl-10">
                    <label class="cursor-pointer flex items-center gap-3">
                        <input type="checkbox" bind:checked={deprovPurgeRepos} class="toggle toggle-error toggle-sm" />
                        <span class="text-sm">Hard Purge Git Repositories (Destructive)</span>
                    </label>
                </div>
            {/if}

            <div class="divider my-1"></div>

            <label class="cursor-pointer flex items-center gap-4">
                <input type="checkbox" checked disabled class="checkbox checkbox-error opacity-50" />
                <span class="font-bold flex-1">Revoke Edge Agent Access</span>
            </label>
            <div class="pl-10">
                <label class="cursor-pointer flex items-center gap-3">
                    <input type="checkbox" bind:checked={deprovPurgeHome} class="toggle toggle-error toggle-sm" />
                    <span class="text-sm">Hard Purge /home Directories (Destructive)</span>
                </label>
            </div>
        </div>
        
        <div class="modal-action mt-6">
            <button class="btn btn-ghost" on:click={() => closeModal('modal_deprovision')}>Cancel</button>
            <button class="btn btn-error" on:click={handleDeprovision} disabled={isProcessing}>
                {#if isProcessing} <span class="loading loading-spinner loading-sm"></span> {/if}
                Confirm Deprovision
            </button>
        </div>
    </div>
</dialog>