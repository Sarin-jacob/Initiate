<script>
    import { createEventDispatcher } from 'svelte';
    import Avatar from '../ui/Avatar.svelte';
    
    export let users = [];
    export let giteaUrl = "";

    const dispatch = createEventDispatcher();
    
    // Filter & Search State
    let searchQuery = "";
    let statusFilter = "ALL";

    // Fixed Dropdown Menu State
    let activeMenu = null; // Holds { user, top, left }

    // Reactive filtered list
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

    // Toggle Dropdown Position relative to Viewport
    function toggleMenu(event, user) {
        event.stopPropagation();
        
        // If clicking the same user's menu, toggle it off
        if (activeMenu?.user.ID === user.ID) {
            activeMenu = null;
            return;
        }

        const rect = event.currentTarget.getBoundingClientRect();
        const menuWidth = 224; // 14rem (w-56)
        
        // Ensure menu doesn't go off the left edge on mobile screens
        const calculatedLeft = Math.max(10, rect.right - menuWidth);

        activeMenu = {
            user,
            top: rect.bottom + window.scrollY + 6,
            left: calculatedLeft
        };
    }

    function closeMenu() {
        activeMenu = null;
    }

    function triggerAction(type, user) {
        closeMenu();
        dispatch('action', { type, user });
    }
</script>

<!-- Automatically close dropdown when scrolling or clicking anywhere outside -->
<svelte:window on:click={closeMenu} on:scroll={closeMenu} />

<div class="card bg-base-100 shadow-sm border border-base-300">
    <!-- Filter Bar -->
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
                <option value="DEPROVISION_FAILED">Failed Teardown</option>
            </select>
        </div>
    </div>

    <!-- Data Table -->
    <div class="overflow-x-auto">
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
                        <td class="text-right">
                            <button 
                                type="button" 
                                class="btn btn-ghost btn-sm btn-circle"
                                on:click={(e) => toggleMenu(e, user)}
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 12.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 18.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z" /></svg>
                            </button>
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

<!-- TELEPORTED VIEWPORT-FIXED DROPDOWN MENU -->
{#if activeMenu}
    <!-- Background overlay stopPropagation so clicking inside menu doesn't close immediately -->
    <ul 
        class="fixed z-50 menu p-2 shadow-2xl bg-base-100 rounded-box w-56 border border-base-300 text-left animate-in fade-in zoom-in-95 duration-100"
        style="top: {activeMenu.top}px; left: {activeMenu.left}px;"
        on:click={(e) => e.stopPropagation()}
        role="menu"
    >
        <li class="menu-title px-4 py-2">Manage {activeMenu.user.Username}</li>
        <li><button type="button" on:click={() => triggerAction('expiry', activeMenu.user)}>Extend Expiration</button></li>
        <li><button type="button" on:click={() => triggerAction('macro', activeMenu.user)}>Apply Manual Macro</button></li>
        <div class="divider my-1"></div>
        <li><button type="button" class="text-error font-bold" on:click={() => triggerAction('deprovision', activeMenu.user)}>Deprovision User</button></li>
    </ul>
{/if}