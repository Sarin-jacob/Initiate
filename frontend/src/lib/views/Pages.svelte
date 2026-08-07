<script>
    import { onMount } from 'svelte';

    let pages = [];
    let currentSlug = '';
    let currentTitle = '';
    let currentContent = '';
    
    let isSaving = false;
    let alertMsg = '';
    let isError = false;

    const cheatSheet = ['{{.Username}}', '{{.Email}}', '{{.GiteaURL}}', '{{.Token}}'];

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-admin'
    };

    async function fetchPages() {
        try {
            const res = await fetch('/api/admin/pages', { headers });
            if (res.ok) pages = await res.json() || [];
        } catch (err) { console.error(err); }
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

    function copyVar(text) {
        navigator.clipboard.writeText(text);
        alertMsg = `Copied ${text} to clipboard!`;
        isError = false;
        setTimeout(() => alertMsg = '', 3000);
    }

    async function handleSave(e) {
        e.preventDefault();
        isSaving = true;
        try {
            const res = await fetch('/api/admin/pages', {
                method: 'POST',
                headers,
                body: JSON.stringify({ slug: currentSlug, title: currentTitle, content: currentContent })
            });
            if (!res.ok) throw new Error("Failed to save page");
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
                            {page.Title}
                            <span class="text-sm opacity-50 block font-mono">/{page.Slug}</span>
                        </button>
                    </li>
                {:else}
                    <li class="disabled"><span>No pages found</span></li>
                {/each}
            </ul>
        </div>
    </div>

    <!-- Main Editor -->
    <div class="lg:col-span-3 card bg-base-100 shadow-sm border border-base-300 flex flex-col text-base-content text-lg">
        <div class="card-body flex flex-col h-full">
            <h2 class="card-title mb-2">{currentSlug ? 'Edit Document' : 'Create New Document'}</h2>

            {#if alertMsg}
                <div class="alert {isError ? 'alert-error' : 'alert-success'} shadow-sm p-3 mb-2"><span>{alertMsg}</span></div>
            {/if}

            <form on:submit={handleSave} class="flex flex-col flex-1 space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">Title</span></label>
                        <input type="text" bind:value={currentTitle} required class="input input-bordered" />
                    </div>
                    <div class="form-control">
                        <label class="label"><span class="label-text font-bold">URL Slug</span></label>
                        <input type="text" bind:value={currentSlug} required class="input input-bordered font-mono" />
                    </div>
                </div>

                <div class="form-control flex-1 flex flex-col">
                    <div class="flex justify-between items-end mb-2">
                        <label class="label p-0"><span class="label-text font-bold">Markdown Content</span></label>
                        <div class="dropdown dropdown-end">
                            <div tabindex="0" role="button" class="btn btn-xs btn-outline">Variables</div>
                            <ul class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-48 text-md font-mono">
                                {#each cheatSheet as item}
                                    <li><a on:click={() => copyVar(item)}>{item}</a></li>
                                {/each}
                            </ul>
                        </div>
                    </div>
                    <textarea bind:value={currentContent} required class="textarea textarea-bordered font-mono flex-1 w-full bg-base-200/30 leading-relaxed resize-none"></textarea>
                </div>

                <div class="card-actions justify-end mt-4">
                    <button type="submit" class="btn btn-primary" disabled={isSaving || !currentSlug}>Save Page</button>
                </div>
            </form>
        </div>
    </div>
</div>