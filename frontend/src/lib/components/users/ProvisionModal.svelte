<script>
    import { createEventDispatcher } from 'svelte';
    const dispatch = createEventDispatcher();

    export let servers = [];
    export let pages = [];
    
    // UI State
    let isBulk = false;
    let isInviting = false;
    let alertMsg = "";
    let progressLog = []; // Tracks bulk success/fails

    // Form State
    let singleUser = { username: '', email: '' };
    let bulkUsers = [{ username: '', email: '' }];
    
    let expireAmount = "";
    let expireUnit = "days";
    let selectedTargets = [];
    let selectedDocs = [];
    
    // Macro State
    let adminFormFields = [];
    let isInspectingVars = false;

    // --- LOGIC ---
    function addBulkRow() {
        bulkUsers = [...bulkUsers, { username: '', email: '' }];
    }

    function removeBulkRow(index) {
        if (bulkUsers.length > 1) {
            bulkUsers = bulkUsers.filter((_, i) => i !== index);
        }
    }

    function toggleDoc(slug) {
        selectedDocs = selectedDocs.includes(slug)
            ? selectedDocs.filter(d => d !== slug)
            : [...selectedDocs, slug];
    }

    function toggleTarget(id) {
        selectedTargets = selectedTargets.includes(id)
            ? selectedTargets.filter(t => t !== id)
            : [...selectedTargets, id];
        updateAdminVars(selectedTargets);
    }

    async function updateAdminVars(targets) {
        if (!targets || targets.length === 0) {
            adminFormFields = [];
            return;
        }

        isInspectingVars = true;
        try {
            const res = await fetch('/api/admin/macros/admin-vars', {
                method: 'POST',
                headers: { 
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + sessionStorage.getItem('nexus_jwt')
                },
                body: JSON.stringify({ target_ids: targets })
            });
            const data = await res.json();
            
            // Preserve old values so they don't wipe when a new server is checked
            const oldValues = {};
            adminFormFields.forEach(f => { oldValues[f.name] = f.value; });
            
            const vars = data.admin_vars || {};
            
            // Transform the dictionary into a safe array of objects
            adminFormFields = Object.entries(vars).map(([name, type]) => ({
                name,
                type,
                value: oldValues[name] || (type === 'bool' || type === 'boolean' ? 'false' : '')
            }));
            
        } catch (err) {
            console.error("Failed to inspect variables", err);
        } finally {
            isInspectingVars = false;
        }
    }

    async function handleSubmit(e) {
        e.preventDefault();
        isInviting = true;
        alertMsg = "";
        progressLog = [];

        const usersToProcess = isBulk ? bulkUsers.filter(u => u.username && u.email) : [singleUser];

        if (usersToProcess.length === 0) {
            alertMsg = "Please provide at least one username and email.";
            isInviting = false; return;
        }
        if (selectedTargets.length === 0) {
            alertMsg = "You must assign at least one Edge Server.";
            isInviting = false; return;
        }
        const finalAdminInputs = {};
        adminFormFields.forEach(f => {
            finalAdminInputs[f.name] = f.value;
        });

        for (const user of usersToProcess) {
            try {
                const payload = {
                    username: user.username.trim(),
                    email: user.email.trim(),
                    expire_amount: parseInt(expireAmount) || 0,
                    expire_unit: expireUnit,
                    target_ids: selectedTargets,
                    admin_context: JSON.stringify(finalAdminInputs),
                    injected_docs: JSON.stringify(selectedDocs)
                };

                const res = await fetch('/api/admin/users/invite', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + sessionStorage.getItem('nexus_jwt')
                    },
                    body: JSON.stringify(payload)
                });

                if (!res.ok) throw new Error(await res.text());
                progressLog = [...progressLog, { user: user.username, status: 'Success' }];
            } catch (err) {
                progressLog = [...progressLog, { user: user.username, status: 'Error', error: err.message }];
            }
        }

        isInviting = false;

        if (progressLog.every(p => p.status === 'Success')) {
            setTimeout(() => {
                document.getElementById('provisionModal').close();
                dispatch('success'); // Tells parent to refresh table
                resetForm();
            }, 1500);
        }
    }

    function resetForm() {
        singleUser = { username: '', email: '' };
        bulkUsers = [{ username: '', email: '' }];
        selectedTargets = [];
        selectedDocs = [];
        adminFormFields = {};
        progressLog = [];
        alertMsg = "";
    }
</script>

<dialog id="provisionModal" class="modal modal-bottom sm:modal-middle" on:close={resetForm}>
    <!-- ADDED: flex flex-col max-h-[90vh] to control the height perfectly -->
    <div class="modal-box w-11/12 max-w-4xl bg-base-100 p-0 overflow-hidden flex flex-col max-h-[90vh]">
        
        <!-- HEADER (Fixed at Top) -->
        <div class="bg-base-200/50 p-6 border-b border-base-200 flex-shrink-0">
            <h3 class="font-bold text-xl text-base-content">Provision Access</h3>
            <p class="text-sm opacity-60">Deploy automated pipelines for new users.</p>
        </div>

        <!-- FORM CONTAINER (Flex to hold body and footer) -->
        <form on:submit={handleSubmit} class="flex flex-col flex-1 overflow-hidden">
            
            <!-- BODY (Scrollable Middle Section) -->
            <div class="p-6 overflow-y-auto flex-1 space-y-8">
                
                {#if alertMsg}
                    <div class="alert alert-error shadow-sm text-sm py-2">{alertMsg}</div>
                {/if}

                <!-- DAISYUI TABS -->
                <div role="tablist" class="tabs tabs-boxed bg-base-200/50 w-full md:w-max">
                    <button type="button" role="tab" class="tab {isBulk ? '' : 'tab-active font-bold'}" on:click={() => isBulk = false}>Single User</button>
                    <button type="button" role="tab" class="tab {isBulk ? 'tab-active font-bold' : ''}" on:click={() => isBulk = true}>Bulk Import</button>
                </div>

                <!-- STEP 1: IDENTITIES -->
                <div class="space-y-4">
                    <h4 class="font-bold text-primary border-b border-base-200 pb-2">1. Identities</h4>
                    
                    {#if !isBulk}
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div class="form-control">
                                <label class="label"><span class="label-text font-bold">Username</span></label>
                                <input type="text" bind:value={singleUser.username} required class="input input-bordered w-full" />
                            </div>
                            <div class="form-control">
                                <label class="label"><span class="label-text font-bold">Email</span></label>
                                <input type="email" bind:value={singleUser.email} required class="input input-bordered w-full" />
                            </div>
                        </div>
                    {:else}
                        <!-- Removed fixed height here so it flows nicely in the parent scroll container -->
                        <div class="space-y-3 bg-base-200/30 p-4 rounded-xl border border-base-200">
                            {#each bulkUsers as user, i}
                                <div class="flex items-center gap-2">
                                    <input type="text" bind:value={user.username} placeholder="Username" required class="input input-sm input-bordered w-full font-mono" />
                                    <input type="email" bind:value={user.email} placeholder="Email" required class="input input-sm input-bordered w-full" />
                                    {#if bulkUsers.length > 1}
                                        <button type="button" class="btn btn-square btn-ghost btn-sm text-error" on:click={() => removeBulkRow(i)}>✕</button>
                                    {/if}
                                </div>
                            {/each}
                            <button type="button" class="btn btn-sm btn-outline btn-primary mt-2" on:click={addBulkRow}>+ Add Row</button>
                        </div>
                    {/if}
                </div>

                <!-- STEP 2: CONFIGURATION -->
                <div class="space-y-4">
                    <h4 class="font-bold text-secondary border-b border-base-200 pb-2">2. Pipeline Configuration</h4>
                    
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <!-- Targets -->
                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Target Servers</span></label>
                            <div class="space-y-2 border border-base-300 p-3 rounded-xl max-h-48 overflow-y-auto bg-base-100">
                                {#each servers as server}
                                    <label class="flex items-center gap-3 p-2 hover:bg-base-200 rounded-lg cursor-pointer transition-colors {selectedTargets.includes(server.ID) ? 'bg-secondary/10 border border-secondary/20' : ''}">
                                        <input type="checkbox" class="checkbox checkbox-sm checkbox-secondary" checked={selectedTargets.includes(server.ID)} on:change={() => toggleTarget(server.ID)} />
                                        <span class="font-bold text-sm">{server.Name}</span>
                                    </label>
                                {/each}
                            </div>
                        </div>

                        <!-- Expiry & Macros -->
                        <div class="space-y-4">
                            <div class="form-control">
                                <label class="label"><span class="label-text font-bold">Expiration</span></label>
                                <div class="join w-full">
                                    <input type="number" bind:value={expireAmount} min="0" placeholder="0 = Never" class="input input-bordered join-item w-full" />
                                    <select bind:value={expireUnit} class="select select-bordered join-item">
                                        <option value="days">Days</option>
                                        <option value="weeks">Weeks</option>
                                        <option value="months">Months</option>
                                    </select>
                                </div>
                            </div>

                            {#if isInspectingVars}
                                <div class="text-sm opacity-50 flex items-center gap-2"><span class="loading loading-spinner loading-xs"></span> Scanning pipelines...</div>
                                
                            {:else if adminFormFields.length > 0}
                                <div class="bg-base-200 p-4 rounded-xl border border-base-300">
                                    <h5 class="text-xs font-bold uppercase tracking-wide mb-2 opacity-70">Required Variables</h5>
                                    
                                    {#each adminFormFields as field}
                                        <div class="form-control mb-2">
                                            <label class="label py-1"><span class="label-text text-sm capitalize">{field.name.replace(/_/g, ' ')}</span></label>
                                            
                                            {#if field.type === 'secret'}
                                                <input type="password" bind:value={field.value} required class="input input-bordered input-sm w-full font-mono" />
                                                
                                            {:else if field.type === 'textarea'}
                                                <textarea bind:value={field.value} required class="textarea textarea-bordered textarea-sm w-full font-mono" rows="2"></textarea>
                                                
                                            {:else if field.type === 'bool' || field.type === 'boolean'}
                                                <select bind:value={field.value} required class="select select-bordered select-sm w-full font-mono">
                                                    <option value="true">True</option>
                                                    <option value="false">False</option>
                                                </select>
                                                
                                            {:else if field.type === 'int' || field.type === 'number'}
                                                <input type="number" bind:value={field.value} required class="input input-bordered input-sm w-full font-mono" />
                                                
                                            {:else}
                                                <input type="text" bind:value={field.value} required class="input input-bordered input-sm w-full font-mono" />
                                            {/if}
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        </div>
                    </div>
                </div>

                <!-- STEP 3: DOCS -->
                <div class="space-y-4">
                    <h4 class="font-bold border-b border-base-200 pb-2">3. Documentation</h4>
                    <div class="flex flex-wrap gap-2">
                        {#each pages as page}
                            <label class="badge badge-lg cursor-pointer gap-2 py-3 {selectedDocs.includes(page.Slug) ? 'badge-primary' : 'badge-outline'}">
                                <input type="checkbox" class="hidden" checked={selectedDocs.includes(page.Slug)} on:change={() => toggleDoc(page.Slug)} />
                                {page.Title}
                            </label>
                        {:else}
                            <div class="text-sm opacity-50 italic">No documentation pages found.</div>
                        {/each}
                    </div>
                </div>

                <!-- Execution Progress Log -->
                {#if progressLog.length > 0}
                    <div class="bg-base-200 p-4 rounded-xl space-y-1 mt-4 font-mono text-sm">
                        {#each progressLog as p}
                            <div class="flex justify-between border-b border-base-300 last:border-0 pb-1 last:pb-0">
                                <span>{p.user}</span>
                                <span class={p.status === 'Success' ? 'text-success' : 'text-error'}>{p.status} {p.error ? `(${p.error})` : ''}</span>
                            </div>
                        {/each}
                    </div>
                {/if}
            </div>

            <!-- FOOTER (Fixed at Bottom) -->
            <div class="bg-base-100 p-6 border-t border-base-200 flex justify-end gap-3 flex-shrink-0 z-10">
                <button type="button" class="btn btn-ghost" on:click={() => document.getElementById('provisionModal').close()}>Cancel</button>
                <button type="submit" class="btn btn-primary px-8" disabled={isInviting}>
                    {#if isInviting} <span class="loading loading-spinner"></span> {/if}
                    Deploy {isBulk ? 'Bulk' : ''} Access
                </button>
            </div>
        </form>
    </div>
    <form method="dialog" class="modal-backdrop"><button>close</button></form>
</dialog>