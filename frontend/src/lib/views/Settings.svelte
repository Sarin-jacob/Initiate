<script>
    import { onMount } from 'svelte';

    // Bound to the parent App.svelte so theme changes reflect instantly
    export let systemTheme; 

    let settings = {
        theme: 'corporate',
        gitea_url: '',
        default_invite_slug: 'index',
        user_expire_days: '0'
    };

    let isLoading = false;
    let isSaving = false;
    let alertMsg = '';
    let isError = false;

    // daisyUI built-in themes to choose from
    const availableThemes = [
        "corporate", "dark", "light", "cupcake", "synthwave", 
        "retro", "cyberpunk", "valentine", "aqua", "night", "winter", "dim"
    ];

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-admin'
    };

    onMount(async () => {
        isLoading = true;
        try {
            const res = await fetch('/api/admin/settings', { headers });
            if (res.ok) {
                const data = await res.json();
                // Merge fetched data into local state
                settings = { ...settings, ...data };
            }
        } catch (err) {
            console.error("Failed to fetch settings", err);
        } finally {
            isLoading = false;
        }
    });

    // Update parent theme immediately when the select dropdown changes
    function handleThemeChange() {
        systemTheme = settings.theme;
    }

    async function handleSave(e) {
        e.preventDefault();
        isSaving = true;
        alertMsg = '';

        try {
            const res = await fetch('/api/admin/settings', {
                method: 'POST',
                headers,
                body: JSON.stringify(settings)
            });
            
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to save settings");
            
            alertMsg = 'System settings updated successfully.';
            isError = false;
        } catch (err) {
            alertMsg = err.message;
            isError = true;
        } finally {
            isSaving = false;
        }
    }
</script>

<div class="max-w-4xl">
    <div class="mb-8">
        <h1 class="text-3xl font-bold">System Settings</h1>
        <p class="text-base-content/70 mt-1">Configure core platform behavior, integrations, and aesthetics.</p>
    </div>

    {#if alertMsg}
        <div class="alert {isError ? 'alert-error' : 'alert-success'} shadow-sm mb-6">
            <span>{alertMsg}</span>
        </div>
    {/if}

    {#if isLoading}
        <div class="flex justify-center p-12">
            <span class="loading loading-spinner loading-lg text-primary"></span>
        </div>
    {:else}
        <form on:submit={handleSave} class="space-y-6">
            
            <!-- Appearance Section -->
            <div class="card bg-base-100 shadow-sm border border-base-300">
                <div class="card-body">
                    <h2 class="card-title text-lg border-b border-base-200 pb-2 mb-4">Appearance</h2>
                    
                    <div class="form-control max-w-md">
                        <label class="label"><span class="label-text font-bold">Global UI Theme</span></label>
                        <select bind:value={settings.theme} on:change={handleThemeChange} class="select select-bordered w-full capitalize">
                            {#each availableThemes as themeOpt}
                                <option value={themeOpt}>{themeOpt}</option>
                            {/each}
                        </select>
                        <label class="label"><span class="label-text-alt text-base-content/60">Changes are previewed instantly.</span></label>
                    </div>
                </div>
            </div>

            <!-- Integrations Section -->
            <div class="card bg-base-100 shadow-sm border border-base-300">
                <div class="card-body">
                    <h2 class="card-title text-lg border-b border-base-200 pb-2 mb-4">Integrations & Architecture</h2>
                    
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Central Gitea Instance URL</span></label>
                            <input type="url" bind:value={settings.gitea_url} required class="input input-bordered w-full" placeholder="http://localhost:3000" />
                            <label class="label"><span class="label-text-alt text-base-content/60">Used to dynamically fetch user avatars.</span></label>
                        </div>

                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Default Onboarding Page (Slug)</span></label>
                            <input type="text" bind:value={settings.default_invite_slug} required class="input input-bordered w-full font-mono" placeholder="index" />
                            <label class="label"><span class="label-text-alt text-base-content/60">The CMS page loaded when a user clicks an invite link.</span></label>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Security & Lifecycle Section -->
            <div class="card bg-base-100 shadow-sm border border-base-300">
                <div class="card-body">
                    <h2 class="card-title text-lg border-b border-base-200 pb-2 mb-4">Security & Lifecycle</h2>
                    
                    <div class="form-control max-w-md">
                        <label class="label"><span class="label-text font-bold">Automated User Expiration (Days)</span></label>
                        <div class="join w-full">
                            <input type="number" bind:value={settings.user_expire_days} min="0" required class="input input-bordered join-item w-full" />
                            <div class="btn btn-neutral join-item pointer-events-none">Days</div>
                        </div>
                        <label class="label"><span class="label-text-alt text-base-content/60">Set to 0 to disable automated system suspension.</span></label>
                    </div>
                </div>
            </div>

            <div class="flex justify-end pt-4">
                <button type="submit" class="btn btn-primary btn-wide shadow-sm" disabled={isSaving}>
                    {#if isSaving} <span class="loading loading-spinner"></span> {/if}
                    Save Configuration
                </button>
            </div>
        </form>
    {/if}
</div>