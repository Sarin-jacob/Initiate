<script>
    import { onMount } from 'svelte';
    import Onboarding from './lib/Onboarding.svelte';
    import Settings from './lib/views/Settings.svelte';
    import Pages from './lib/views/Pages.svelte';
    import Users from './lib/views/Users.svelte';
    import Agents from './lib/views/Agents.svelte';
    import Macros from './lib/views/Macros.svelte';

    const urlParams = new URLSearchParams(window.location.search);
    const inviteToken = urlParams.get('token');

    let currentView = 'users'; 
    let isAppReady = false;
    
    // Read from localStorage initially
    let systemTheme = localStorage.getItem('nexus_theme') || 'corporate';

    // SVELTE REACTIVITY: Whenever systemTheme changes, update the DOM and localStorage instantly
    $: {
        if (typeof window !== 'undefined') {
            localStorage.setItem('nexus_theme', systemTheme);
            document.documentElement.setAttribute('data-theme', systemTheme);
        }
    }

    onMount(async () => {
        if (inviteToken) {
            isAppReady = true;
            return;
        }

        // Background sync: Fetch global settings to ensure we are aligned with the server
        try {
            const res = await fetch('/api/admin/settings', {
                headers: { 'Authorization': 'Bearer test-admin' }
            });
            if (res.ok) {
                const data = await res.json();
                if (data.theme && data.theme !== systemTheme) {
                    systemTheme = data.theme; // This triggers the reactive statement above!
                }
            }
        } catch (err) {
            console.error("Failed to load global settings", err);
        } finally {
            isAppReady = true;
        }
    });
</script>

<!-- We no longer need data-theme on this div, as it is handled on the <html> tag -->
<div class="min-h-screen bg-base-200 text-base-content font-sans text-lg">
    {#if !isAppReady}
        <div class="flex items-center justify-center min-h-screen">
            <span class="loading loading-spinner loading-lg text-primary"></span>
        </div>
    {:else if inviteToken}
        <Onboarding token={inviteToken} />
    {:else}
        <!-- Admin Dashboard Layout -->
        <div class="drawer lg:drawer-open">
            <input id="admin-drawer" type="checkbox" class="drawer-toggle" />
            
            <div class="drawer-content flex flex-col">
                
                <!-- UNIVERSAL TOPBAR (Desktop & Mobile) -->
                <div class="navbar bg-base-100 shadow-sm border-b border-base-300 px-4">
                    <div class="flex-none lg:hidden">
                        <label for="admin-drawer" class="btn btn-square btn-ghost">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="inline-block w-6 h-6 stroke-current"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
                        </label>
                    </div>
                    
                    <div class="flex-1 lg:hidden px-2 mx-2 font-bold text-xl tracking-tight">NexusIAM</div>
                    <div class="flex-1 hidden lg:block"></div> <!-- Spacer for desktop -->

                    <!-- Settings Cog (Top Right) -->
                    <div class="flex-none">
                        <div class="tooltip tooltip-left" data-tip="System Settings">
                            <button class="btn btn-square btn-ghost {currentView === 'settings' ? 'bg-base-200' : ''}" on:click={() => currentView = 'settings'}>
                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" />
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
                                </svg>
                            </button>
                        </div>
                    </div>
                </div>
                
                <!-- Main Content Area -->
                <main class="p-6 md:p-10 max-w-7xl mx-auto w-full">
                    {#if currentView === 'settings'} <Settings bind:systemTheme={systemTheme} /> {/if}
                    {#if currentView === 'users'} <Users /> {/if}
                    {#if currentView === 'agents'} <Agents /> {/if}
                    {#if currentView === 'macros'} <Macros /> {/if}
                    {#if currentView === 'pages'} <Pages /> {/if}
                </main>
            </div> 
            
            <!-- Desktop/Mobile Sidebar -->
            <div class="drawer-side border-r border-base-300 shadow-xl z-50">
                <label for="admin-drawer" class="drawer-overlay"></label> 
                <ul class="menu p-6 w-72 min-h-full bg-base-100 text-base-content flex flex-col gap-2 text-lg">
                    <li class="mb-6 mt-2 px-4 flex-row items-center gap-3 pointer-events-none">
                        <div class="w-10 h-10 rounded-xl bg-primary text-primary-content flex items-center justify-center font-bold text-xl shadow-sm">N</div>
                        <span class="text-2xl font-bold tracking-tight">NexusIAM</span>
                    </li>
                    
                    <li><a class={currentView === 'users' ? 'active' : ''} on:click={() => currentView = 'users'}>Users & Access</a></li>
                    <li><a class={currentView === 'agents' ? 'active' : ''} on:click={() => currentView = 'agents'}>Edge Agents</a></li>
                    <li><a class={currentView === 'macros' ? 'active' : ''} on:click={() => currentView = 'macros'}>Provisioning Macros</a></li>
                    <li><a class={currentView === 'pages' ? 'active' : ''} on:click={() => currentView = 'pages'}>CMS & Guides</a></li>
                </ul>
            </div>
        </div>
    {/if}
</div>