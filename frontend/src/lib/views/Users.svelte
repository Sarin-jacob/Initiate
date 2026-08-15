<script>
    import { onMount, onDestroy } from 'svelte';
    import UserTable from '../components/users/UserTable.svelte';
    import ActionModals from '../components/users/ActionModals.svelte';
    import ProvisionModal from '../components/users/ProvisionModal.svelte'
    
    let users = [];
    let servers = [];
    let macros = []; 
    let pages = [];
    let giteaUrl = ""; 
    let pollInterval;
    
    let modalsRef; // Reference to our ActionModals component

    const getHeaders = () => ({ 
            'Content-Type': 'application/json', 
            'Authorization': 'Bearer ' + sessionStorage.getItem('nexus_jwt') 
        });

    async function fetchData() {
        try {
            // Added settings fetch to grab the Gitea URL for Avatars
            const [resUsers, resServers, resMacros, resPages, resSettings] = await Promise.all([
                fetch('/api/admin/users', { headers: getHeaders() }),
                fetch('/api/admin/servers', { headers: getHeaders() }),
                fetch('/api/admin/macros', { headers: getHeaders() }),
                fetch('/api/admin/pages', { headers: getHeaders() }),
                fetch('/api/admin/settings', { headers: getHeaders() })
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

    onMount(()=>{
        fetchData();
        pollInterval = setInterval(() => {
            fetchData();
        }, 60000);
    });

    onDestroy(() => {
        if (pollInterval) clearInterval(pollInterval);
    });

    // Handles the custom event dispatched by UserTable.svelte
    function handleAction(event) {
        const { type, user } = event.detail;
        modalsRef.open(type, user); // Calls the exported function inside ActionModals
    }
</script>

<div class="container mx-auto p-4 md:p-8 max-w-7xl">
    
    <!-- Top Action Bar -->
    <div class="flex justify-between items-center mb-8">
        <div>
            <h1 class="text-3xl font-bold">User Management</h1>
            <p class="text-sm opacity-60">Manage identities, access pipelines, and edge deployments.</p>
        </div>
        
        <!-- Trigger Button for the new Modal -->
        <button 
            class="btn btn-primary shadow-sm"
            on:click={() => document.getElementById('provisionModal').showModal()}
        >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 1 1-6.75 0 3.375 3.375 0 0 1 6.75 0ZM4 19.235v-.11a6.375 6.375 0 0 1 12.75 0v.109A12.318 12.318 0 0 1 10.374 21c-2.331 0-4.512-.645-6.374-1.766Z" /></svg>
            Provision Access
        </button>
    </div>

    <!-- The Data Table -->
    <UserTable {users} {giteaUrl} on:action={handleAction} on:refresh={fetchData} />

    <!-- The Isolated Component -->
    <ProvisionModal {servers} {pages} on:success={fetchData} />

</div>
<ActionModals bind:this={modalsRef} {macros} on:refresh={fetchData}/>