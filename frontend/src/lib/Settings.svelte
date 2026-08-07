<script>
    import { onMount } from 'svelte';

    // State
    let pages = [];
    let currentSlug = '';
    let currentTitle = '';
    let currentContent = '';
    
    let isSaving = false;
    let alertMsg = '';
    let isError = false;

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-admin'
    };

    async function fetchPages() {
        try {
            const res = await fetch('/api/admin/pages', { headers });
            if (res.ok) {
                const data = await res.json();
                pages = data || [];
            }
        } catch (err) {
            console.error("Failed to fetch pages", err);
        }
    }

    onMount(fetchPages);

    function selectPage(page) {
        currentSlug = page.Slug;
        currentTitle = page.Title;
        currentContent = page.Content;
        alertMsg = '';
    }

    function createNewPage() {
        currentSlug = '';
        currentTitle = '';
        currentContent = '# New Document\n\nWrite your markdown here...';
        alertMsg = '';
    }

    async function handleSave(e) {
        e.preventDefault();
        isSaving = true;
        alertMsg = '';

        try {
            const res = await fetch('/api/admin/pages', {
                method: 'POST',
                headers,
                body: JSON.stringify({
                    slug: currentSlug,
                    title: currentTitle,
                    content: currentContent
                })
            });
            
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Failed to save page");
            
            alertMsg = 'Page saved successfully!';
            isError = false;
            await fetchPages();
        } catch (err) {
            alertMsg = err.message;
            isError = true;
        } finally {
            isSaving = false;
        }
    }
</script>

<div class="grid grid-cols-1 lg:grid-cols-4 gap-6">
    
    <!-- Sidebar: Page List -->
    <div class="card bg-base-100 shadow-xl border border-base-300 h-fit">
        <div class="card-body p-4">
            <div class="flex justify-between items-center mb-4">
                <h2 class="card-title text-lg">Documents</h2>
                <button class="btn btn-sm btn-circle btn-ghost" on:click={createNewPage} title="New Page">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
                </button>
            </div>
            
            <ul class="menu bg-base-200 rounded-box w-full">
                {#each pages as page}
                    <li>
                        <button 
                            class={currentSlug === page.Slug ? 'active font-bold' : ''} 
                            on:click={() => selectPage(page)}>
                            {page.Title}
                            <span class="text-xs opacity-50 block font-mono">/{page.Slug}</span>
                        </button>
                    </li>
                {:else}
                    <li class="disabled"><span>No pages found</span></li>
                {/each}
            </ul>
        </div>
    </div>

    <!-- Main Editor Area -->
    <div class="lg:col-span-3 card bg-base-100 shadow-xl border border-base-300">
        <div class="card-body">
            <h2 class="card-title mb-4">
                {currentSlug ? 'Edit Document' : 'Create New Document'}
            </h2>

            {#if alertMsg}
                <div class="alert {isError ? 'alert-error' : 'alert-success'} shadow-sm mb-4 p-3">
                    <span>{alertMsg}</span>
                </div>
            {/if}

            <form on:submit={handleSave} class="space-y-4">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Page Title</span></label>
                        <input type="text" bind:value={currentTitle} required class="input input-bordered" placeholder="e.g. SSH Setup Guide" />
                    </div>
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">URL Slug</span></label>
                        <input type="text" bind:value={currentSlug} required class="input input-bordered font-mono" placeholder="e.g. ssh-guide" />
                        <label class="label"><span class="label-text-alt">Must be unique (e.g., 'index', 'welcome')</span></label>
                    </div>
                </div>

                <!-- Markdown Editor -->
                <div class="form-control">
                    <div class="flex justify-between items-end">
                        <label class="label"><span class="label-text font-bold">Markdown Content</span></label>
                        <div class="dropdown dropdown-end">
                            <div tabindex="0" role="button" class="btn btn-xs btn-outline m-1">Variables Cheat Sheet</div>
                            <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-64 border border-base-300 text-sm font-mono">
                                <li><a>{`{{.Username}}`}</a></li>
                                <li><a>{`{{.Email}}`}</a></li>
                                <li><a>{`{{.GiteaURL}}`}</a></li>
                                <li><a>{`{{.Token}}`}</a></li>
                            </ul>
                        </div>
                    </div>
                    <textarea bind:value={currentContent} required class="textarea textarea-bordered font-mono h-96 w-full leading-relaxed bg-base-200/50" placeholder="## Write your content here..."></textarea>
                    
                    <!-- Example linking instruction -->
                    <label class="label">
                        <span class="label-text-alt text-info flex items-center gap-1">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                            Tip: Link to other pages using [Link Text](/api/invite/{`{{.Token}}`}/page/your-slug)
                        </span>
                    </label>
                </div>

                <div class="card-actions justify-end mt-4">
                    <button type="submit" class="btn btn-primary" disabled={isSaving || !currentSlug}>
                        {#if isSaving} <span class="loading loading-spinner"></span> {/if}
                        Save Page
                    </button>
                </div>
            </form>
        </div>
    </div>
</div>