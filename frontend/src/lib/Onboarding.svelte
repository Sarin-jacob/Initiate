<script>
    import { onMount } from 'svelte';
    export let token = "";

    // Base Invite State
    let inviteData = null;
    let fetchError = '';
    
    // Form State
    let isLoading = false;
    let submitMsg = '';
    let isSuccess = false;

    // NEW: Dynamic Form State
    let formFields = [];

    // CMS / Multi-page State
    let currentHtml = '';
    let currentTitle = 'Account Setup';
    let isPageLoading = false;
    let viewingCustomPage = false; 
    let originalHtml = '';

    onMount(async () => {
        try {
            const res = await fetch(`/api/invite/${token}`);
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Invalid or expired token");
            
            inviteData = data;
            currentHtml = data.html_content;
            originalHtml = data.html_content;

            const vars = data.required_vars || {};
            
            // Map the Go dictionary into a safe array of objects
            formFields = Object.entries(vars).map(([name, type]) => ({
                name,
                type,
                value: (type === 'bool' || type === 'boolean' ? 'false' : ''),
                confirmValue: ''
            }));

        } catch (err) {
            fetchError = err.message; 
        }
    });

    async function loadPage(slug) {
        isPageLoading = true;
        try {
            const res = await fetch(`/api/invite/${token}/page/${slug}`);
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Page not found");

            currentTitle = data.title;
            currentHtml = data.html_content;
            viewingCustomPage = true;
        } catch (err) {
            alert("Error loading page: " + err.message);
        } finally {
            isPageLoading = false;
        }
    }

    function handleMarkdownClick(e) {
        const a = e.target.closest('a');
        if (!a) return;

        const href = a.getAttribute('href');
        if (!href) return;

        const pageMatch = href.match(/\/api\/invite\/[^/]+\/page\/(.+)/);
        if (pageMatch) {
            e.preventDefault(); 
            const slug = pageMatch[1];
            loadPage(slug);
        }
    }

    function backToWelcome() {
        currentTitle = 'Account Setup';
        currentHtml = originalHtml;
        viewingCustomPage = false;
    }

    async function handleSetup(e) {
        e.preventDefault();
        submitMsg = '';
        for (const f of formFields) {
            if (f.type === 'secret' && f.value !== f.confirmValue) {
                submitMsg = `Passwords do not match for "${f.name.replace(/_/g, ' ')}". Please try again.`;
                return; // Abort submission
            }
        }
        
        isLoading = true;
        // Rebuild dictionary for Go Backend
        const finalUserInputs = {};
        formFields.forEach(f => {
            finalUserInputs[f.name] = f.value;
        });

        try {
            const res = await fetch(`/api/invite/${token}/complete`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ user_inputs: finalUserInputs })
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || res.statusText);
            
            isSuccess = true;
            submitMsg = 'Provisioning Complete! You can now close this window and log into your services.';
        } catch (err) {
            submitMsg = err.message;
        } finally {
            isLoading = false;
        }
    }

</script>

<div class="container mx-auto p-6 max-w-3xl flex flex-col justify-center min-h-[80vh]">
    {#if fetchError}
        <div class="alert alert-error shadow-lg">
            <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            <span><strong>Access Denied:</strong> {fetchError}</span>
        </div>
    {:else if !inviteData}
        <div class="flex flex-col items-center justify-center space-y-4 text-gray-500">
            <span class="loading loading-ring loading-lg"></span>
            <p>Verifying secure token...</p>
        </div>
    {:else}
        
        {#if isSuccess}
            <div class="card bg-success text-success-content shadow-xl">
                <div class="card-body items-center text-center py-16">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                    <h2 class="card-title text-3xl mb-2">You're all set!</h2>
                    <p class="text-lg">{submitMsg}</p>
                </div>
            </div>
        {:else}
            <div class="card bg-base-100 shadow-xl border border-base-300 overflow-hidden transition-all duration-300">
                <!-- Top Header Section -->
                <div class="bg-primary text-primary-content p-8 text-center relative">
                    {#if viewingCustomPage}
                        <button on:click={backToWelcome} class="btn btn-sm btn-ghost text-primary-content absolute left-4 top-4 opacity-80 hover:opacity-100 hover:text-accent">
                            &larr; Back to Welcome
                        </button>
                    {/if}
                    <h1 class="text-3xl font-bold mb-2">{currentTitle}</h1>
                    {#if !viewingCustomPage}
                        <p class="opacity-90">Hello {inviteData.username}, complete your profile below.</p>
                    {/if}
                </div>

                <div class="card-body p-8">
                    <!-- Dynamic Markdown Renderer -->
                    <div 
                        class="prose prose-sm md:prose-base prose-a:text-primary prose-a:underline prose-a:font-bold hover:prose-a:text-primary-focus max-w-none mb-8 bg-base-200 p-6 rounded-xl relative"
                        on:click={handleMarkdownClick} 
                        role="article" 
                        tabindex="0" 
                        on:keydown={(e) => e.key === 'Enter' && handleMarkdownClick(e)}
                    >
                        {#if isPageLoading}
                            <div class="absolute inset-0 bg-base-200/80 flex items-center justify-center rounded-xl z-10 backdrop-blur-sm">
                                <span class="loading loading-spinner loading-lg text-primary"></span>
                            </div>
                        {/if}
                        
                        <!-- Use {@html} cautiously - we sanitize this using bluemonday on the Go backend! -->
                        {@html currentHtml}
                    </div>

                    {#if submitMsg}
                        <div class="alert alert-error mb-6">
                            <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                            <span>{submitMsg}</span>
                        </div>
                    {/if}

                    <div class="divider">Finalize Provisioning</div>

                    <form on:submit={handleSetup} class="space-y-6">
                        
                        <!-- DYNAMIC FORM GENERATOR -->
                        {#each formFields as field}
                            <div class="form-control">
                                <label class="label"><span class="label-text font-bold capitalize">{field.name.replace(/_/g, ' ')}</span></label>
                                
                                {#if field.type === 'secret'}
                                    <input type="password" bind:value={field.value} required class="input input-bordered input-primary w-full" placeholder="Enter {field.name.replace(/_/g, ' ')}" />
                                    <label class="label mt-2"><span class="label-text font-bold capitalize">Confirm {field.name.replace(/_/g, ' ')}</span></label>
                                    <input type="password" bind:value={field.confirmValue} required class="input input-bordered input-primary w-full" placeholder="Confirm {field.name.replace(/_/g, ' ')}" />
                                    
                                {:else if field.type === 'textarea'}
                                    <textarea bind:value={field.value} required rows="4" class="textarea textarea-bordered textarea-primary font-mono text-sm leading-relaxed"></textarea>
                                    
                                {:else if field.type === 'bool' || field.type === 'boolean'}
                                    <select bind:value={field.value} class="select select-bordered select-primary w-full" required>
                                        <option value="true">True</option>
                                        <option value="false">False</option>
                                    </select>
                                    
                                {:else if field.type === 'int' || field.type === 'number'}
                                    <input type="number" bind:value={field.value} required class="input input-bordered input-primary w-full" />
                                    
                                {:else}
                                    <input type="text" bind:value={field.value} required class="input input-bordered input-primary w-full" />
                                {/if}
                            </div>
                        {:else}
                            <div class="alert alert-success bg-success/10 text-sm mb-6 border border-success/20">
                                No additional credentials required. Click below to execute your automated provisioning pipeline.
                            </div>
                        {/each}

                        <button type="submit" class="btn btn-primary btn-block btn-lg mt-4 shadow-sm" disabled={isLoading}>
                            {#if isLoading} <span class="loading loading-spinner"></span> {/if}
                            Activate My Account
                        </button>
                    </form>
                </div>
            </div>
        {/if}
        
    {/if}
</div>