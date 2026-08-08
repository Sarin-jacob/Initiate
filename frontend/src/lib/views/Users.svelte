<script>
    import { onMount } from 'svelte';
    
    let users = [];
    let servers = [];
    let macros = []; 
    let pages = []; // NEW: CMS Pages for docs
    
    // UI State for the simplified invite form
    let selectedTargets = []; // Array of Server IDs
    let selectedDocs = [];    // Array of Page Slugs
    
    let activeUser = null; 
    let isProcessing = false;
    let isInviting = false;
    let alertMsg = '';

    // Modals
    let deprovPurgeRepos = false;
    let deprovPurgeHome = false;
    let selectedMacroId = '';
    let selectedMacroServer = '';
    let updateExpiryAmount = 0;
    let updateExpiryUnit = 'days';
    
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchData() {
        try {
            const [resUsers, resServers, resMacros, resPages] = await Promise.all([
                fetch('/api/admin/users', { headers }),
                fetch('/api/admin/servers', { headers }),
                fetch('/api/admin/macros', { headers }),
                fetch('/api/admin/pages', { headers })
            ]);
            
            if (resUsers.ok) users = await resUsers.json() || [];
            if (resServers.ok) servers = await resServers.json() || [];
            if (resMacros.ok) macros = await resMacros.json() || [];
            if (resPages.ok) pages = await resPages.json() || [];
        } catch (err) { console.error("Failed to load data", err); }
    }

    onMount(fetchData);

    function formatExpiry(dateStr) {
        if (!dateStr) return "Never";
        const d = new Date(dateStr);
        if (d < new Date()) return "Expired";
        return d.toLocaleDateString();
    }

    function toggleTarget(id) {
        if (selectedTargets.includes(id)) {
            selectedTargets = selectedTargets.filter(t => t !== id);
        } else {
            selectedTargets = [...selectedTargets, id];
        }
    }

    function toggleDoc(slug) {
        if (selectedDocs.includes(slug)) {
            selectedDocs = selectedDocs.filter(d => d !== slug);
        } else {
            selectedDocs = [...selectedDocs, slug];
        }
    }

    async function handleInvite(e) {
        e.preventDefault();
        if (selectedTargets.length === 0) return alert("Select at least one target system.");
        
        isInviting = true;
        const form = e.target;
        
        // The backend now just takes the IDs, it will look up the Agent's configured macros itself
        const payload = {
            username: form.username.value,
            email: form.email.value,
            expire_amount: parseInt(form.expireAmount.value) || 0,
            expire_unit: form.expireUnit.value,
            target_ids: selectedTargets,
            doc_slugs: selectedDocs // Instructs backend to append these links to the welcome email
        };

        try {
            const res = await fetch('/api/admin/users/invite', { method: 'POST', headers, body: JSON.stringify(payload) });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to invite user");
            
            alertMsg = "User invited successfully!";
            form.reset();
            selectedTargets = [];
            selectedDocs = [];
            fetchData();
        } catch (err) { alertMsg = err.message; }
        finally { isInviting = false; }
    }

    function openModal(modalId, user) {
        activeUser = user;
        document.getElementById(modalId).showModal();
    }
    function closeModal(modalId) {
        activeUser = null;
        document.getElementById(modalId).close();
    }

    // Modal Actions (Same logic as before, sending generic payloads)
    async function handleExtendExpiry() {
        isProcessing = true;
        try {
            await fetch(`/api/admin/users/${activeUser.ID}/expire`, {
                method: 'PUT', headers,
                body: JSON.stringify({ expire_amount: parseInt(updateExpiryAmount) || 0, expire_unit: updateExpiryUnit })
            });
            closeModal('modal_expiry');
            fetchData();
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
            closeModal('modal_macro');
            fetchData();
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
    }

    async function handleDeprovision() {
        if (!confirm(`WARNING: Are you sure you want to deprovision ${activeUser.Username}?`)) return;
        isProcessing = true;
        
        // We pass the booleans. The backend will choose Soft vs Hard macro based on these.
        const payload = {
            purge_repos: deprovPurgeRepos,
            purge_home: deprovPurgeHome
        };

        try {
            const res = await fetch(`/api/admin/users/${activeUser.ID}/deprovision`, {
                method: 'POST', headers, body: JSON.stringify(payload)
            });
            if (!res.ok) throw new Error("Deprovisioning failed (user may be logged in)");
            closeModal('modal_deprovision');
            fetchData();
        } catch (err) { alert(err.message); }
        finally { isProcessing = false; }
    }
</script>

<div class="space-y-8">
    <div>
        <h1 class="text-4xl font-bold">Users & Access</h1>
        <p class="text-base-content/70 mt-2 text-lg">Manage identities and orchestrate lifecycle pipelines.</p>
    </div>

    <!-- PROVISIONING ACCORDION -->
    <div class="collapse collapse-arrow bg-base-100 border border-base-300 shadow-sm">
        <input type="checkbox" /> 
        <div class="collapse-title text-xl font-bold p-6">+ Provision New User Access</div>
        <div class="collapse-content border-t border-base-200 p-6">
            <form on:submit={handleInvite} class="space-y-6 pt-4">
                {#if alertMsg}<div class="alert shadow-sm mb-4">{alertMsg}</div>{/if}
                
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
                
                <div class="form-control max-w-md">
                    <label class="label"><span class="label-text font-bold">Automated Expiration</span></label>
                    <div class="join w-full">
                        <input type="number" name="expireAmount" min="0" placeholder="0 = Never" class="input input-bordered join-item w-full input-lg" />
                        <select name="expireUnit" class="select select-bordered join-item input-lg">
                            <option value="days">Days</option>
                            <option value="weeks">Weeks</option>
                            <option value="months">Months</option>
                            <option value="years">Years</option>
                        </select>
                    </div>
                </div>

                <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 mt-8 border-t border-base-200 pt-8">
                    <!-- TARGET SELECTION -->
                    <div>
                        <h3 class="font-bold text-lg mb-4">1. Grant Access To:</h3>
                        <div class="space-y-3">
                            {#each servers as server}
                                <label class="flex items-center gap-4 p-4 border border-base-300 rounded-xl cursor-pointer hover:bg-base-200/50 transition-colors {selectedTargets.includes(server.ID) ? 'border-primary bg-primary/5' : 'bg-base-100'}">
                                    <input type="checkbox" class="checkbox checkbox-primary" checked={selectedTargets.includes(server.ID)} on:change={() => toggleTarget(server.ID)} />
                                    <div>
                                        <div class="font-bold">{server.Name}</div>
                                        <div class="text-sm opacity-60">
                                            {server.ID === 'internal-gitea' ? 'Virtual System' : 'Edge Agent ' + server.ID.substring(0,8)}
                                        </div>
                                    </div>
                                </label>
                            {/each}
                        </div>
                    </div>

                    <!-- DOCUMENTATION INJECTION -->
                    <div>
                        <h3 class="font-bold text-lg mb-4">2. Include Documentation:</h3>
                        <p class="text-sm opacity-70 mb-4">Selected guides will be linked in the user's welcome email.</p>
                        <div class="space-y-2 max-h-64 overflow-y-auto">
                            {#each pages as page}
                                <label class="flex items-center gap-3 p-3 border border-base-200 rounded-lg cursor-pointer hover:bg-base-200/50 {selectedDocs.includes(page.Slug) ? 'bg-base-200' : ''}">
                                    <input type="checkbox" class="checkbox checkbox-sm" checked={selectedDocs.includes(page.Slug)} on:change={() => toggleDoc(page.Slug)} />
                                    <span class="font-mono text-sm">{page.Title}</span>
                                </label>
                            {:else}
                                <div class="text-sm opacity-50 p-4 border border-dashed rounded-lg">No CMS pages created yet.</div>
                            {/each}
                        </div>
                    </div>
                </div>
                
                <div class="pt-4 flex justify-end">
                    <button type="submit" class="btn btn-primary btn-lg px-12" disabled={isInviting}>
                        {#if isInviting} <span class="loading loading-spinner"></span> {/if}
                        Generate & Send Invite
                    </button>
                </div>
            </form>
        </div>
    </div>

    <!-- IDENTITY MATRIX TABLE -->
    <div class="card bg-base-100 shadow-sm border border-base-300">
        <div class="overflow-x-auto">
            <table class="table table-zebra w-full text-base">
                <thead class="bg-base-200 text-base">
                    <tr><th>Identity</th><th>Status</th><th>Expires</th><th>Granted Access</th><th>Actions</th></tr>
                </thead>
                <tbody>
                    {#each users as user}
                        <tr>
                            <td>
                                <div class="font-bold text-lg">{user.Username}</div>
                                <div class="text-sm opacity-60">{user.Email}</div>
                            </td>
                            <td>
                                <!-- Safe Failure Badge Display -->
                                {#if user.Status === 'DEPROVISION_FAILED'}
                                    <span class="badge badge-error p-3 tooltip" data-tip="Requires manual SSH cleanup">Failed Teardown</span>
                                {:else if user.Status === 'ACTIVE'}
                                    <span class="badge badge-success p-3">Active</span>
                                {:else}
                                    <span class="badge badge-warning p-3">{user.Status}</span>
                                {/if}
                            </td>
                            <td>
                                <span class="text-sm font-mono {formatExpiry(user.ExpiresAt) === 'Expired' ? 'text-error font-bold' : ''}">
                                    {formatExpiry(user.ExpiresAt)}
                                </span>
                            </td>
                            <td>
                                <div class="flex flex-wrap gap-2">
                                    {#if user.access_list}
                                        {#each user.access_list as srv}
                                            <span class="badge {srv.TargetID === 'internal-gitea' ? 'badge-secondary' : 'badge-info'} p-3" title={srv.TargetID}>
                                                {srv.TargetID === 'internal-gitea' ? 'Gitea' : srv.TargetID.substring(0, 8)}
                                            </span>
                                        {/each}
                                    {/if}
                                </div>
                            </td>
                            <td class="w-16">
                                <div class="dropdown dropdown-end">
                                    <div tabindex="0" role="button" class="btn btn-ghost btn-sm btn-circle">
                                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 12.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 18.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z" /></svg>
                                    </div>
                                    <ul class="dropdown-content z-[1] menu p-2 shadow-lg bg-base-100 rounded-box w-56 border border-base-300">
                                        <li class="menu-title px-4 py-2">Manage {user.Username}</li>
                                        <li><button type="button" on:click|preventDefault={() => openModal('modal_expiry', user)}>Extend Expiration</button></li>
                                        <li><button type="button" on:click|preventDefault={() => openModal('modal_macro', user)}>Apply Manual Macro</button></li>
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

<!-- ================= MODALS ================= -->

<!-- Expiry Modal -->
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

<!-- Manual Macro Modal -->
<dialog id="modal_macro" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Run Manual Macro Pipeline</h3>
        <p class="text-sm opacity-70 mb-4">Execute a standalone sequence of actions for <b>{activeUser?.Username}</b>.</p>
        
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
            <button class="btn btn-ghost" on:click={() => closeModal('modal_macro')}>Cancel</button>
            <button class="btn btn-primary" on:click={handleApplyMacro} disabled={isProcessing || !selectedMacroId || !selectedMacroServer}>
                {#if isProcessing} <span class="loading loading-spinner loading-sm"></span> {/if}
                Execute Pipeline
            </button>
        </div>
    </div>
</dialog>

<!-- Deprovision Modal -->
<dialog id="modal_deprovision" class="modal">
    <div class="modal-box border-t-4 border-error">
        <h3 class="font-bold text-lg text-error mb-2">Deprovision {activeUser?.Username}</h3>
        <p class="text-sm opacity-70 mb-6">This will execute the assigned Deprovisioning Macros for every target this user has access to.</p>
        
        <div class="space-y-4 bg-base-200/50 p-4 rounded-xl border border-base-300">
            <p class="text-sm font-bold opacity-70 mb-2 uppercase">Destructive Purge Flags</p>
            <label class="cursor-pointer flex items-center gap-3">
                <input type="checkbox" bind:checked={deprovPurgeRepos} class="toggle toggle-error toggle-sm" />
                <span class="text-sm">Purge Git Repositories (Passed to Gitea Macros)</span>
            </label>
            <label class="cursor-pointer flex items-center gap-3">
                <input type="checkbox" bind:checked={deprovPurgeHome} class="toggle toggle-error toggle-sm" />
                <span class="text-sm">Purge /home Directories (Passed to Server Macros)</span>
            </label>
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