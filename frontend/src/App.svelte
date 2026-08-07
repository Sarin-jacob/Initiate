<script>
    import { onMount } from 'svelte';
    import Onboarding from './lib/Onboarding.svelte';
    import Settings from './lib/views/Settings.svelte';
    import Users from './lib/views/Users.svelte';
    import Agents from './lib/views/Agents.svelte';
    import Pages from './lib/views/Pages.svelte';

    const urlParams = new URLSearchParams(window.location.search);
    const inviteToken = urlParams.get('token');

    let currentView = 'settings'; // Defaulting to settings for testing
    let systemTheme = 'corporate'; // Fallback theme
    let isAppReady = false;

    onMount(async () => {
        // If they are a standard user onboarding, skip admin fetching
        if (inviteToken) {
            isAppReady = true;
            return;
        }

        // Fetch global system settings for the Admin App Shell
        try {
            const res = await fetch('/api/admin/settings', {
                headers: { 'Authorization': 'Bearer test-admin' }
            });
            if (res.ok) {
                const data = await res.json();
                if (data.theme) systemTheme = data.theme;
            }
        } catch (err) {
            console.error("Failed to load global settings", err);
        } finally {
            isAppReady = true;
        }
    });
</script>

<!-- The data-theme attribute controls daisyUI colors globally -->
<div data-theme={systemTheme} class="min-h-screen bg-base-200 text-base-content font-sans text-lg">
    {#if !isAppReady}
        <div class="flex items-center justify-center min-h-screen">
            <span class="loading loading-spinner loading-lg text-primary"></span>
        </div>
    {:else if inviteToken}
        <!-- Locked Onboarding Flow -->
        <Onboarding token={inviteToken} />
    {:else}
        <!-- Admin Dashboard Layout (Sidebar + Content) -->
        <div class="drawer lg:drawer-open">
            <input id="admin-drawer" type="checkbox" class="drawer-toggle" />
            
            <div class="drawer-content flex flex-col">
                <!-- Mobile Header -->
                <div class="w-full navbar bg-base-100 lg:hidden shadow-sm">
                    <div class="flex-none">
                        <label for="admin-drawer" class="btn btn-square btn-ghost">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="inline-block w-6 h-6 stroke-current"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
                        </label>
                    </div>
                    <div class="flex-1 px-2 mx-2 font-bold text-xl tracking-tight">NexusIAM</div>
                </div>
                
                <!-- Main View Area -->
                <main class="p-6 md:p-10 max-w-7xl mx-auto w-full">
                    {#if currentView === 'settings'} 
                        <!-- We pass the theme down so the Settings component can update it live -->
                        <Settings bind:systemTheme={systemTheme} /> 
                    {/if}
                    
                    {#if currentView === 'users'} <Users /> {/if}
                    {#if currentView === 'agents'} <Agents /> {/if}
                    {#if currentView === 'pages'} <Pages /> {/if}
                   
                </main>
            </div> 
            
            <!-- Desktop/Mobile Sidebar -->
            <div class="drawer-side border-r border-base-300 shadow-xl z-50">
                <label for="admin-drawer" class="drawer-overlay"></label> 
                <ul class="menu p-4 w-64 min-h-full bg-base-100 text-base-content flex flex-col gap-2 text-lg">
                    <li class="mb-6 mt-2 px-4 flex-row items-center gap-3 pointer-events-none">
                        <div class="w-8 h-8 rounded-lg bg-primary text-primary-content flex items-center justify-center font-bold">N</div>
                        <span class="text-xl font-bold tracking-tight">NexusIAM</span>
                    </li>
                    
                    <li><a class={currentView === 'users' ? 'active' : ''} on:click={() => currentView = 'users'}>Users & Access</a></li>
                    <li><a class={currentView === 'agents' ? 'active' : ''} on:click={() => currentView = 'agents'}>Edge Agents</a></li>
                    <li><a class={currentView === 'pages' ? 'active' : ''} on:click={() => currentView = 'pages'}>CMS & Guides</a></li>
                    
                    <div class="divider my-2">System</div>
                    
                    <li><a class={currentView === 'settings' ? 'active' : ''} on:click={() => currentView = 'settings'}>System Settings</a></li>
                </ul>
            </div>
        </div>
    {/if}
</div>