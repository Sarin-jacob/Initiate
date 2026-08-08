<script>
    import { onMount } from 'svelte';
    import { addToast } from '../stores/toast.js'; // Global Notifications

    export let systemTheme; 

    let settings = {
        theme: 'corporate',
        gitea_url: '',
        default_invite_slug: 'default-onboarding',
        default_email_slug: 'default-email', // NEW
        user_expire_days: '0'
    };

    let pages = []; // For template dropdowns
    let isLoading = false;
    let isSaving = false;

    const availableThemes = [
        "corporate", "dark", "light", "cupcake", "synthwave", 
        "retro", "cyberpunk", "valentine", "aqua", "night", "winter", "dim"
    ];

    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    onMount(async () => {
        isLoading = true;
        try {
            const [setRes, pageRes] = await Promise.all([
                fetch('/api/admin/settings', { headers }),
                fetch('/api/admin/pages', { headers })
            ]);
            if (setRes.ok) {
                const data = await setRes.json();
                settings = { ...settings, ...data };
            }
            if (pageRes.ok) pages = await pageRes.json();
        } catch (err) {
            addToast("Failed to load settings from server", "error");
        } finally {
            isLoading = false;
        }
    });

    function handleThemeChange() {
        systemTheme = settings.theme;
    }

    async function handleSave(e) {
        e.preventDefault();
        isSaving = true;
        try {
            const res = await fetch('/api/admin/settings', {
                method: 'POST', headers, body: JSON.stringify(settings)
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to save settings");
            
            addToast("System settings updated successfully", "success");
        } catch (err) {
            addToast(err.message, "error");
        } finally {
            isSaving = false;
        }
    }
</script>

<div class="max-w-4xl space-y-8">
    <div>
        <h1 class="text-4xl font-bold">System Settings</h1>
        <p class="text-base-content/70 mt-2 text-lg">Configure core platform behavior, integrations, and aesthetics.</p>
    </div>

    {#if isLoading}
        <div class="flex justify-center p-12">
            <span class="loading loading-spinner loading-lg text-primary"></span>
        </div>
    {:else}
        <form on:submit={handleSave} class="space-y-8">
            
            <!-- 1. APPEARANCE -->
            <div class="card bg-base-100 shadow-sm border border-base-300">
                <div class="card-body">
                    <h2 class="card-title text-xl border-b border-base-200 pb-2 mb-4">Appearance</h2>
                    <div class="form-control max-w-md">
                        <label class="label"><span class="label-text font-bold">Global UI Theme</span></label>
                        <select bind:value={settings.theme} on:change={handleThemeChange} class="select select-bordered w-full capitalize">
                            {#each availableThemes as themeOpt}
                                <option value={themeOpt}>{themeOpt}</option>
                            {/each}
                        </select>
                        <label class="label"><span class="label-text-alt opacity-70">Changes are previewed instantly across the dashboard.</span></label>
                    </div>
                </div>
            </div>

            <!-- 2. SYSTEM TEMPLATES -->
            <div class="card bg-base-100 shadow-sm border border-base-300">
                <div class="card-body">
                    <h2 class="card-title text-xl border-b border-base-200 pb-2 mb-4">System Templates</h2>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Default Welcome Email</span></label>
                            <select bind:value={settings.default_email_slug} class="select select-bordered w-full font-mono">
                                {#each pages as page}
                                    <option value={page.Slug}>{page.Title} (/{page.Slug})</option>
                                {/each}
                            </select>
                            <label class="label"><span class="label-text-alt opacity-70">The email dispatched with the token link.</span></label>
                        </div>

                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Default Onboarding UI</span></label>
                            <select bind:value={settings.default_invite_slug} class="select select-bordered w-full font-mono">
                                {#each pages as page}
                                    <option value={page.Slug}>{page.Title} (/{page.Slug})</option>
                                {/each}
                            </select>
                            <label class="label"><span class="label-text-alt opacity-70">The page loaded when a user clicks their invite link.</span></label>
                        </div>
                    </div>
                </div>
            </div>

            <!-- 3. INTEGRATIONS & LIFECYCLE -->
            <div class="card bg-base-100 shadow-sm border border-base-300">
                <div class="card-body">
                    <h2 class="card-title text-xl border-b border-base-200 pb-2 mb-4">Integrations & Lifecycle</h2>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Central Gitea Instance URL</span></label>
                            <input type="url" bind:value={settings.gitea_url} class="input input-bordered w-full" placeholder="http://localhost:3000" />
                            <label class="label"><span class="label-text-alt opacity-70">Used to fetch avatars dynamically.</span></label>
                        </div>

                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Automated User Expiration</span></label>
                            <div class="join w-full">
                                <input type="number" bind:value={settings.user_expire_days} min="0" required class="input input-bordered join-item w-full" />
                                <div class="btn btn-neutral join-item pointer-events-none">Days</div>
                            </div>
                            <label class="label"><span class="label-text-alt opacity-70">Set to 0 to disable automated suspension.</span></label>
                        </div>
                    </div>
                </div>
            </div>

            <div class="flex justify-end pt-2 pb-10">
                <button type="submit" class="btn btn-primary btn-lg px-12 shadow-sm" disabled={isSaving}>
                    {#if isSaving} <span class="loading loading-spinner"></span> {/if}
                    Save System Configuration
                </button>
            </div>
        </form>
    {/if}
</div>