<script>
    let isLoading = false;
    let alertMsg = '';
    let isError = false;

    let defaultMarkdown = `## Welcome **{{.Username}}**!
Please set your password below to finalize your system access. Your Gitea account will be registered under \`{{.Email}}\`.`;

    async function handleInvite(e) {
        e.preventDefault();
        isLoading = true;
        alertMsg = '';

        const form = e.target;
        const payload = {
            username: form.username.value,
            email: form.email.value,
            provision_gitea: form.provGitea.checked,
            edge_server_ids: form.servers.value.split(',').map(s => s.trim()).filter(Boolean),
            markdown_template: form.markdown.value
        };

        try {
            const res = await fetch('/api/admin/users/invite', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer test-admin' },
                body: JSON.stringify(payload)
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || res.statusText);
            
            alertMsg = 'Success! Check the server logs for the invite link.';
            isError = false;
            form.reset();
        } catch (err) {
            alertMsg = err.message;
            isError = true;
        } finally {
            isLoading = false;
        }
    }
</script>

<div class="container mx-auto p-6 max-w-4xl">
    <div class="flex justify-between items-center mb-8">
        <h1 class="text-3xl font-bold">NexusIAM Admin</h1>
        <div class="badge badge-primary badge-outline">System Online</div>
    </div>

    {#if alertMsg}
        <div class="alert {isError ? 'alert-error' : 'alert-success'} shadow-sm mb-6">
            <span>{alertMsg}</span>
        </div>
    {/if}

    <div class="card bg-base-100 shadow-xl border border-base-300">
        <div class="card-body">
            <h2 class="card-title text-2xl mb-4">Provision New Access</h2>
            
            <form on:submit={handleInvite} class="space-y-4">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div class="form-control">
                        <label class="label"><span class="label-text font-semibold">Username</span></label>
                        <input type="text" name="username" required class="input input-bordered w-full" placeholder="jdoe" />
                    </div>
                    <div class="form-control">
                        <label class="label"><span class="label-text font-semibold">Email</span></label>
                        <input type="email" name="email" required class="input input-bordered w-full" placeholder="user@company.com" />
                    </div>
                </div>

                <div class="form-control">
                    <label class="label"><span class="label-text font-semibold">Target Edge Server IDs</span></label>
                    <input type="text" name="servers" class="input input-bordered w-full" placeholder="test-pc-001, prod-web-02" value="test-pc-001" />
                    <label class="label"><span class="label-text-alt text-gray-500">Comma separated IDs of registered agents</span></label>
                </div>

                <div class="form-control bg-base-200 p-4 rounded-lg">
                    <label class="label cursor-pointer justify-start gap-4">
                        <input type="checkbox" name="provGitea" class="checkbox checkbox-primary" checked />
                        <span class="label-text font-bold">Provision Central Gitea Account</span>
                    </label>
                </div>

                <div class="form-control">
                    <label class="label"><span class="label-text font-semibold">Onboarding Guide (Markdown)</span></label>
                    <textarea name="markdown" bind:value={defaultMarkdown} class="textarea textarea-bordered font-mono h-32" required></textarea>
                </div>

                <div class="card-actions justify-end mt-6">
                    <button type="submit" class="btn btn-primary w-full md:w-auto" disabled={isLoading}>
                        {#if isLoading} <span class="loading loading-spinner"></span> {/if}
                        Send Invitation
                    </button>
                </div>
            </form>
        </div>
    </div>
</div>