<script>
    import { onMount } from 'svelte';
    import { addToast } from '../stores/toast.js';

    let pages = [];
    let settings = {};
    
    // Editor State
    let currentSlug = '';
    let currentTitle = '';
    let currentContent = '';
    
    // Preview State
    let activeTab = 'edit'; // 'edit' | 'preview'
    let previewHtml = '';
    let isPreviewLoading = false;
    
    let isSaving = false;
    let isDeleting = false;

    const cheatSheet = ['{{.Username}}', '{{.Email}}', '{{.GiteaURL}}', '{{.Token}}', '{{.InviteURL}}'];
    const headers = { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' };

    async function fetchData() {
        try {
            const [pgRes, setRes] = await Promise.all([
                fetch('/api/admin/pages', { headers }),
                fetch('/api/admin/settings', { headers })
            ]);
            if (pgRes.ok) pages = await pgRes.json() || [];
            if (setRes.ok) settings = await setRes.json() || {};
        } catch (err) { addToast("Failed to fetch CMS data", "error"); }
    }

    onMount(fetchData);

    function selectPage(page) {
        currentSlug = page.Slug;
        currentTitle = page.Title;
        currentContent = page.Content;
        activeTab = 'edit';
    }

    function createNewPage() {
        currentSlug = '';
        currentTitle = '';
        currentContent = '# New Document\n\nWrite your markdown here...';
        activeTab = 'edit';
    }

    function copyVar(text) {
        navigator.clipboard.writeText(text);
        addToast(`Copied ${text} to clipboard!`, "success", 2000);
    }

    // Toggle Preview and fetch HTML from the Go backend
    async function togglePreview() {
        if (activeTab === 'preview') {
            activeTab = 'edit';
            return;
        }
        
        activeTab = 'preview';
        isPreviewLoading = true;
        
        try {
            const res = await fetch('/api/admin/pages/preview', {
                method: 'POST', headers,
                body: JSON.stringify({ content: currentContent })
            });
            const data = await res.json();
            if (!res.ok) throw new Error("Failed to render preview");
            previewHtml = data.html_content;
        } catch (err) {
            previewHtml = `<div class="text-error font-bold">${err.message}</div>`;
        } finally {
            isPreviewLoading = false;
        }
    }

    async function handleSave(e) {
        e.preventDefault();
        isSaving = true;
        try {
            const res = await fetch('/api/admin/pages', {
                method: 'POST', headers,
                body: JSON.stringify({ slug: currentSlug, title: currentTitle, content: currentContent })
            });
            if (!res.ok) throw new Error("Failed to save document");
            
            addToast("Document saved successfully!", "success");
            await fetchData();
        } catch (err) { addToast(err.message, "error"); }
        finally { isSaving = false; }
    }

    async function handleDelete() {
        if (!confirm(`WARNING: Are you sure you want to delete /${currentSlug}?`)) return;
        isDeleting = true;
        try {
            const res = await fetch(`/api/admin/pages/${currentSlug}`, { method: 'DELETE', headers });
            if (!res.ok) throw new Error("Failed to delete document");
            
            addToast("Document deleted.", "success");
            createNewPage();
            await fetchData();
        } catch (err) { addToast(err.message, "error"); }
        finally { isDeleting = false; }
    }
</script>

<div class="grid grid-cols-1 lg:grid-cols-4 gap-6 h-[calc(100vh-8rem)] text-base-content text-lg">
    
    <!-- Sidebar: Page List -->
    <div class="card bg-base-100 shadow-sm border border-base-300 h-full">
        <div class="card-body p-4 flex flex-col">
            <div class="flex justify-between items-center mb-4">
                <h2 class="card-title text-lg">Documents</h2>
                <button class="btn btn-sm btn-circle btn-ghost bg-base-200" on:click={createNewPage} title="New Page">+</button>
            </div>
            
            <ul class="menu bg-base-200/50 rounded-box w-full flex-1 overflow-y-auto">
                {#each pages as page}
                    <li>
                        <button class={currentSlug === page.Slug ? 'active font-bold' : ''} on:click={() => selectPage(page)}>
                            <div class="flex flex-col text-left w-full">
                                <span>{page.Title}</span>
                                <span class="text-xs opacity-60 font-mono">/{page.Slug}</span>
                                
                                <!-- Visual Badges for System Defaults -->
                                {#if settings.default_invite_slug === page.Slug}
                                    <span class="badge badge-primary badge-xs mt-1 py-1">Onboarding Default</span>
                                {/if}
                                {#if settings.default_email_slug === page.Slug}
                                    <span class="badge badge-secondary badge-xs mt-1 py-1">Email Default</span>
                                {/if}
                            </div>
                        </button>
                    </li>
                {:else}
                    <li class="disabled"><span class="opacity-50">No documents found</span></li>
                {/each}
            </ul>
        </div>
    </div>

    <!-- Main Editor -->
    <div class="lg:col-span-3 card bg-base-100 shadow-sm border border-base-300 flex flex-col text-base-content text-lg">
        <div class="card-body flex flex-col h-full">
            <h2 class="card-title mb-4">{currentSlug ? 'Edit Document' : 'Create New Document'}</h2>

            <form on:submit={handleSave} class="flex flex-col flex-1 space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Title</span></label>
                        <input type="text" bind:value={currentTitle} required class="input input-bordered w-full" />
                    </div>
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">URL Slug</span></label>
                        <input type="text" bind:value={currentSlug} required class="input input-bordered w-full font-mono" />
                    </div>
                </div>

                <div class="form-control flex-1 flex flex-col min-h-0">
                    <div class="flex justify-between items-end mb-2">
                        <div class="tabs tabs-boxed bg-base-200">
                            <button type="button" class="tab {activeTab === 'edit' ? 'tab-active font-bold' : ''}" on:click={() => activeTab = 'edit'}>Write</button>
                            <button type="button" class="tab {activeTab === 'preview' ? 'tab-active font-bold' : ''}" on:click={togglePreview}>Preview HTML</button>
                        </div>
                        
                        <div class="dropdown dropdown-end">
                            <div tabindex="0" role="button" class="btn btn-xs btn-outline">Insert Variable</div>
                            <ul class="dropdown-content z-[1] menu p-2 shadow-xl bg-base-100 rounded-box w-48 text-md font-mono border border-base-300">
                                {#each cheatSheet as item}
                                    <li><a on:click={() => copyVar(item)}>{item}</a></li>
                                {/each}
                            </ul>
                        </div>
                    </div>
                    
                    {#if activeTab === 'edit'}
                        <textarea bind:value={currentContent} required class="textarea textarea-bordered font-mono flex-1 w-full bg-base-200/30 leading-relaxed resize-none"></textarea>
                    {:else}
                        <div class="flex-1 w-full border border-base-300 rounded-xl bg-base-100 overflow-y-auto p-8 relative">
                            {#if isPreviewLoading}
                                <div class="absolute inset-0 bg-base-100/50 backdrop-blur-sm flex items-center justify-center">
                                    <span class="loading loading-spinner text-primary"></span>
                                </div>
                            {/if}
                            <div class="prose prose-sm md:prose-base max-w-none">
                                {@html previewHtml}
                            </div>
                        </div>
                    {/if}
                </div>

                <div class="card-actions justify-between mt-4">
                    {#if currentSlug}
                        <button type="button" class="btn btn-error btn-outline" on:click={handleDelete} disabled={isDeleting}>
                            {#if isDeleting} <span class="loading loading-spinner loading-sm"></span> {/if} Delete
                        </button>
                    {:else}
                        <div></div> <!-- Spacer -->
                    {/if}
                    
                    <button type="submit" class="btn btn-primary px-8" disabled={isSaving || !currentSlug}>
                        {#if isSaving} <span class="loading loading-spinner loading-sm"></span> {/if} Save Document
                    </button>
                </div>
            </form>
        </div>
    </div>
</div>