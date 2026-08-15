<script>
    import { onMount } from 'svelte';
    import Toast from './lib/components/ui/Toast.svelte';
    import Onboarding from './lib/Onboarding.svelte';
    import PublicDoc from './lib/PublicDoc.svelte';
    
    import Settings from './lib/views/Settings.svelte';
    import Pages from './lib/views/Pages.svelte';
    import Users from './lib/views/Users.svelte';
    import Agents from './lib/views/Agents.svelte';
    import Macros from './lib/views/Macros.svelte';
    import Login from './lib/Login.svelte';

    const urlParams = new URLSearchParams(window.location.search);
    const inviteToken = urlParams.get('token');
    const docSlug = urlParams.get('docs');

    // FIX: Initialize directly from sessionStorage to beat the reactive statement!
    let currentView = sessionStorage.getItem('nexus_active_view') || 'users'; 
    let isAppReady = false;
    let isAuthenticated = false; 

    let systemTheme = localStorage.getItem('nexus_theme') || 'corporate';
    let cmsDocs = []; 

    // Apply theme (Keep this in localStorage so theme persists across browser restarts)
    $: {
        if (typeof window !== 'undefined') {
            localStorage.setItem('nexus_theme', systemTheme);
            document.documentElement.setAttribute('data-theme', systemTheme);
        }
    }

    // Safely save the current view whenever it changes
    $: if (currentView) {
        sessionStorage.setItem('nexus_active_view', currentView);
    }

    onMount(async () => {
        // 1. External Public Routes
        if (inviteToken || docSlug) {
            isAppReady = true;
            return;
        }

        // 2. Admin Routes: Check sessionStorage instead of localStorage
        const token = sessionStorage.getItem('nexus_jwt');
        if (token) {
            await loadAdminData(token);
        } else {
            isAppReady = true;
        }
    });

    async function loadAdminData(token) {
        try {
            const headers = { 'Authorization': 'Bearer ' + token };
            
            const [setRes, pagesRes] = await Promise.all([
                fetch('/api/admin/settings', { headers }),
                fetch('/api/admin/pages', { headers })
            ]);
            
            if (setRes.ok) {
                isAuthenticated = true; 
                const data = await setRes.json();
                if (data.theme && data.theme !== systemTheme) systemTheme = data.theme;
                if (pagesRes.ok) cmsDocs = await pagesRes.json() || [];
            } else {
                // Token invalid: Clear from sessionStorage
                sessionStorage.removeItem('nexus_jwt');
                isAuthenticated = false;
            }
        } catch (err) { 
            console.error("Global fetch failed", err); 
        } finally { 
            isAppReady = true; 
        }
    }

    function onLoginSuccess() {
        isAppReady = false; 
        // Read from sessionStorage after successful login
        const token = sessionStorage.getItem('nexus_jwt');
        loadAdminData(token);
    }

    function navigate(view) {
        currentView = view;
        const drawer = document.getElementById('admin-drawer');
        if (drawer) drawer.checked = false; 
    }
</script>

<Toast />

<div class="min-h-screen bg-base-200 text-base-content font-sans">
    {#if !isAppReady}
        <div class="p-10 space-y-4 max-w-7xl mx-auto mt-10">
            <div class="skeleton h-12 w-64"></div>
            <div class="skeleton h-64 w-full"></div>
            <div class="skeleton h-32 w-full"></div>
        </div>
    
    {:else if inviteToken}
        <Onboarding token={inviteToken} />
        
    {:else if docSlug}
        <PublicDoc slug={docSlug} />
        
    {:else if !isAuthenticated}
        <Login on:success={onLoginSuccess} />
        
    {:else}
        <div class="drawer lg:drawer-open">
            <input id="admin-drawer" type="checkbox" class="drawer-toggle" />
            <div class="drawer-content flex flex-col">
                
                <div class="navbar bg-base-100 shadow-sm border-b border-base-300 px-4">
                    <div class="flex-none lg:hidden">
                        <label for="admin-drawer" class="btn btn-square btn-ghost">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="inline-block w-6 h-6 stroke-current"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
                        </label>
                    </div>
                    
                    <div class="flex-1 lg:hidden px-2 mx-2 font-bold text-xl tracking-tight">Nexus</div>
                    <div class="flex-1 hidden lg:block"></div>
                    
                    <div class="flex-none gap-2">
                        <div class="dropdown dropdown-end align-middle">
                            <div tabindex="0" role="button" class="btn btn-ghost flex gap-2">
                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 0 0 6 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 0 1 6 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 0 1 6-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0 0 18 18a8.967 8.967 0 0 0-6 2.292m0-14.25v14.25" /></svg>
                                <span class="hidden md:inline">Docs</span>
                            </div>
                            <ul class="dropdown-content z-1 menu p-2 shadow-lg bg-base-100 rounded-box w-64 border border-base-300">
                                <li class="menu-title px-4 py-2">Quick Reference</li>
                                {#each cmsDocs as doc}
                                    <li><a href="?docs={doc.Slug}" target="_blank" class="font-mono text-sm">{doc.Title}</a></li>
                                {:else}
                                    <li class="disabled"><span class="opacity-50">No docs published</span></li>
                                {/each}
                            </ul>
                        </div>

                        <div class="tooltip tooltip-left" data-tip="System Settings">
                            <button class="btn btn-square btn-ghost {currentView === 'settings' ? 'bg-base-200' : ''}" on:click={() => navigate('settings')}>
                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6"><path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                            </button>
                        </div>
                    </div>
                </div>
                
                <main class="p-6 md:p-10 max-w-7xl mx-auto w-full">
                    {#if currentView === 'settings'} <Settings bind:systemTheme={systemTheme} /> {/if}
                    {#if currentView === 'users'} <Users /> {/if}
                    {#if currentView === 'agents'} <Agents /> {/if}
                    {#if currentView === 'macros'} <Macros /> {/if}
                    {#if currentView === 'pages'} <Pages /> {/if}
                </main>
            </div> 
            
            <div class="drawer-side border-r border-base-300 shadow-xl z-50">
                <label for="admin-drawer" class="drawer-overlay"></label> 
                <ul class="menu p-6 w-72 min-h-full bg-base-100 text-base-content flex flex-col gap-2">
                    <li class="mb-6 mt-2 px-4 flex-row items-center gap-3 pointer-events-none">
                        <div class="w-10 h-10 rounded-xl bg-primary text-primary-content flex items-center justify-center font-bold text-xl shadow-sm p-0">
                            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="34" viewBox="0 0 24 24" fill="none" stroke="currentcolor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user-round-key-icon lucide-user-round-key"><path d="M19 11v6"/><path d="M19 13h2"/><path d="M2 21a8 8 0 0 1 12.868-6.349"/><circle cx="10" cy="8" r="5"/><circle cx="19" cy="19" r="2"/></svg>
                        </div>
                        <span class="text-2xl font-bold tracking-tight">Nexus</span>
                    </li>
                    <li><button class={currentView === 'users' ? 'active font-bold' : ''} on:click={() => navigate('users')}>Users & Access</button></li>
                    <li><button class={currentView === 'agents' ? 'active font-bold' : ''} on:click={() => navigate('agents')}>Edge Agents</button></li>
                    <li><button class={currentView === 'macros' ? 'active font-bold' : ''} on:click={() => navigate('macros')}>Provisioning Macros</button></li>
                    <li><button class={currentView === 'pages' ? 'active font-bold' : ''} on:click={() => navigate('pages')}>CMS & Guides</button></li>
                </ul>
            </div>
        </div>
    {/if}
</div>