<script>
    import { onMount } from 'svelte';
    import InviteForm from '../components/users/InviteForm.svelte';
    import UserTable from '../components/users/UserTable.svelte';
    import ActionModals from '../components/users/ActionModals.svelte';
    
    let users = [];
    let servers = [];
    let macros = []; 
    let pages = [];
    let giteaUrl = ""; 
    
    let modalsRef; // Reference to our ActionModals component

    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchData() {
        try {
            // Added settings fetch to grab the Gitea URL for Avatars
            const [resUsers, resServers, resMacros, resPages, resSettings] = await Promise.all([
                fetch('/api/admin/users', { headers }),
                fetch('/api/admin/servers', { headers }),
                fetch('/api/admin/macros', { headers }),
                fetch('/api/admin/pages', { headers }),
                fetch('/api/admin/settings', { headers })
            ]);
            
            if (resUsers.ok) users = await resUsers.json() || [];
            if (resServers.ok) servers = await resServers.json() || [];
            if (resMacros.ok) macros = await resMacros.json() || [];
            if (resPages.ok) pages = await resPages.json() || [];
            if (resSettings.ok) {
                const settings = await resSettings.json();
                giteaUrl = settings.gitea_url || "";
            }
        } catch (err) { console.error("Failed to load data", err); }
    }

    onMount(fetchData);

    // Handles the custom event dispatched by UserTable.svelte
    function handleAction(event) {
        const { type, user } = event.detail;
        modalsRef.open(type, user); // Calls the exported function inside ActionModals
    }
</script>

<div class="space-y-8">
    <div>
        <h1 class="text-4xl font-bold">Users & Access</h1>
        <p class="text-base-content/70 mt-2 text-lg">Manage identities, search profiles, and orchestrate lifecycle pipelines.</p>
    </div>

    <!-- The Provisioning UI Component -->
    <InviteForm 
        {servers} 
        {macros} 
        {pages} 
        on:refresh={fetchData} 
    />

    <!-- The Data Table Component (with Avatars and Filters) -->
    <UserTable 
        {users} 
        {giteaUrl} 
        on:action={handleAction} 
    />
</div>

<!-- The Modals Container -->
<ActionModals 
    bind:this={modalsRef} 
    {macros} 
    on:refresh={fetchData} 
/>