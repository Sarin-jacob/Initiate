<script>
    import { createEventDispatcher } from 'svelte';
    import Avatar from '../ui/Avatar.svelte';
    
    export let users = [];
    export let giteaUrl = "";

    const dispatch = createEventDispatcher();
    
    let searchQuery = "";
    let statusFilter = "ALL";

    // Log Modal State
    let selectedLog = "";
    let selectedLogUser = "";
    let isLogModalOpen = false;

    $: filteredUsers = users.filter(user => {
        const matchesSearch = user.Username.toLowerCase().includes(searchQuery.toLowerCase()) || 
                              user.Email.toLowerCase().includes(searchQuery.toLowerCase());
        const matchesStatus = statusFilter === "ALL" || user.Status === statusFilter;
        return matchesSearch && matchesStatus;
    });

    function formatExpiry(dateStr) {
        if (!dateStr) return "Never";
        const d = new Date(dateStr);
        if (d < new Date()) return "Expired";
        return d.toLocaleDateString();
    }

    function triggerAction(type, user) {
        // Find and close the open DaisyUI dropdown manually by blurring active element
        if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
        }
        dispatch('action', { type, user });
    }

    function openLogs(user) {
        selectedLogUser = user.Username;
        
        if (user.access_list && user.access_list.length > 0) {
            selectedLog = user.access_list.map(a => {
                const statusStr = a.Status === 'FAILED' ? '[❌ FAILED]' : '[✅ SUCCESS]';
                const logData = a.ExecutionLog || "No logs recorded.";
                return `=== TARGET: ${a.TargetID} ${statusStr} ===\n${logData}`;
            }).join('\n\n');
        } else {
            selectedLog = "No access records found for this user.";
        }
        
        isLogModalOpen = true;
    }
    function hasFailedAccess(user) {
        if (user.Status === 'DEPROVISION_FAILED') return true;
        return user.access_list && user.access_list.some(a => a.Status === 'FAILED');
    }

    async function forceRemoveUser(user) {
        if (!confirm(`Are you sure you want to FORCE REMOVE ${user.Username}? This deletes the database record entirely.`)) {
            return;
        }

        // Close the popover menu
        if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
        }

        try {
            const res = await fetch(`/api/admin/users/${user.ID}/force`, {
                method: 'DELETE',
                headers: { 
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + localStorage.getItem('nexus_jwt') 
                }
            });

            if (!res.ok) {
                const data = await res.json();
                throw new Error(data.message || "Failed to purge user");
            }

            // Immediately remove the user from the local array to update the UI instantly
            users = users.filter(u => u.ID !== user.ID);
            
            // Dispatch an optional refresh just in case the parent wants to update stats
            dispatch('refresh'); 

        } catch (err) {
            alert("Error: " + err.message);
        }
    }
</script>

<!-- LOG VIEWER MODAL -->
{#if isLogModalOpen}
    <div class="modal modal-open">
        <div class="modal-box w-11/12 max-w-4xl bg-base-200">
            <h3 class="font-bold text-lg mb-4">Pipeline Execution Logs: {selectedLogUser}</h3>
            <div class="bg-black text-green-400 font-mono text-sm p-4 rounded-xl h-96 overflow-y-auto whitespace-pre-wrap">
                {selectedLog}
            </div>
            <div class="modal-action">
                <button class="btn btn-primary" on:click={() => isLogModalOpen = false}>Close</button>
            </div>
        </div>
    </div>
{/if}

<div class="card bg-base-100 shadow-sm border border-base-300">
    <div class="p-4 border-b border-base-200 bg-base-200/30 flex flex-col md:flex-row gap-4 items-center justify-between">
        <div class="form-control w-full md:max-w-xs">
            <input type="text" placeholder="Search users..." bind:value={searchQuery} class="input input-sm input-bordered w-full" />
        </div>
        <div class="flex gap-2 w-full md:w-auto">
            <select bind:value={statusFilter} class="select select-sm select-bordered flex-1 md:w-48">
                <option value="ALL">All Statuses</option>
                <option value="ACTIVE">Active</option>
                <option value="PENDING">Pending</option>
                <option value="ARCHIVED">Archived</option>
                <option value="FAILED">Provision Failed</option>
            </select>
        </div>
    </div>

    <!-- ADDED: min-h-[16rem] prevents the dropdown from being clipped if table is short -->
    <div class="overflow-x-auto min-h-[16rem]">
        <table class="table table-zebra w-full text-base">
            <thead class="bg-base-200 text-base">
                <tr><th>Identity</th><th>Status</th><th>Expires</th><th>Granted Access</th><th class="text-right">Actions</th></tr>
            </thead>
            <tbody>
                {#each filteredUsers as user}
                    <tr>
                        <td>
                            <div class="flex items-center gap-4">
                                <Avatar username={user.Username} giteaUrl={giteaUrl} sizeClass="w-12 h-12" />
                                <div>
                                    <div class="font-bold text-lg">{user.Username}</div>
                                    <div class="text-sm opacity-60">{user.Email}</div>
                                </div>
                            </div>
                        </td>
                        <td>
                            {#if hasFailedAccess(user)}
                                <span class="badge badge-error p-3 cursor-pointer hover:bg-error-focus" on:click={() => openLogs(user)}>View Fail Log</span>
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
                            <div class="flex flex-wrap gap-2 max-w-[250px]">
                                {#if user.access_list}
                                    {#each user.access_list as srv}
                                        <span class="badge {srv.TargetID === 'internal-gitea' ? 'badge-secondary' : 'badge-info'} p-3" title={srv.TargetID}>
                                            {srv.TargetID === 'internal-gitea' ? 'Gitea' : srv.TargetID.substring(0, 8)}
                                        </span>
                                    {/each}
                                {/if}
                            </div>
                        </td>
                        <td class="text-right flex justify-end gap-2">
                            {#if user.ExecutionLog}
                                <button class="btn btn-ghost btn-sm btn-circle tooltip tooltip-left" data-tip="View Logs" on:click={() => openLogs(user)}>
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m3.75 9v6m3-3H9m1.5-12H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" /></svg>
                                </button>
                            {/if}
                            
                            <!-- DAISYUI POPOVER DROPDOWN -->
                            <button 
                                class="btn btn-ghost btn-sm btn-circle" 
                                popovertarget="menu-{user.ID}" 
                                style="anchor-name: --anchor-{user.ID}"
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 12.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 18.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z" /></svg>
                            </button>

                            <ul 
                                class="dropdown menu w-56 rounded-box bg-base-100 shadow-xl border border-base-300 text-left mt-2"
                                popover 
                                id="menu-{user.ID}" 
                                style="position-anchor: --anchor-{user.ID}"
                            >
                                <li class="menu-title px-4 py-2">Manage {user.Username}</li>
                                <li><button type="button" on:click={() => triggerAction('expiry', user)}>Extend Expiration</button></li>
                                <li><button type="button" on:click={() => triggerAction('macro', user)}>Apply Manual Macro</button></li>
                                <div class="divider my-1"></div>
                                <li><button type="button" class="text-warning font-bold" on:click={() => triggerAction('deprovision', user)}>Deprovision User</button></li>
                                <li><button type="button" class="text-error font-bold" on:click={() => forceRemoveUser(user)}>Force Remove (DB)</button></li>
                            </ul>
                        </td>
                    </tr>
                {:else}
                    <tr>
                        <td colspan="5" class="text-center p-8 opacity-50">No users found matching your filters.</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
</div>