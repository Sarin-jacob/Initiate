<script>
    import { onMount } from 'svelte';
    export let token = "";

    let inviteData = null;
    let fetchError = '';
    
    let isLoading = false;
    let submitMsg = '';
    let isSuccess = false;

    onMount(async () => {
        try {
            const res = await fetch(`/api/invite/${token}`);
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || "Invalid or expired token");
            inviteData = data;
        } catch (err) {
            fetchError = err.message;
        }
    });

    async function handleSetup(e) {
        e.preventDefault();
        isLoading = true;
        submitMsg = '';

        const form = e.target;
        try {
            const res = await fetch(`/api/invite/${token}/complete`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    password: form.password.value,
                    ssh_public_key: form.sshKey.value
                })
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
                <div class="card-body items-center text-center">
                    <h2 class="card-title text-3xl mb-2">You're all set!</h2>
                    <p>{submitMsg}</p>
                </div>
            </div>
        {:else}
            <div class="card bg-base-100 shadow-xl border border-base-300 overflow-hidden">
                <!-- Top Header Section -->
                <div class="bg-primary text-primary-content p-8 text-center">
                    <h1 class="text-3xl font-bold mb-2">Account Setup</h1>
                    <p class="opacity-90">Hello {inviteData.username}, complete your profile below.</p>
                </div>

                <div class="card-body p-8">
                    <!-- Rendered Markdown from Go Server -->
                    <div class="prose max-w-none mb-8 bg-base-200 p-6 rounded-xl">
                        {@html inviteData.html_content}
                    </div>

                    {#if submitMsg}
                        <div class="alert alert-error mb-6">{submitMsg}</div>
                    {/if}

                    <form on:submit={handleSetup} class="space-y-6">
                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">Set Permanent Password</span></label>
                            <input type="password" name="password" required minlength="8" class="input input-bordered input-primary w-full" placeholder="Must be at least 8 characters" />
                        </div>

                        <div class="form-control">
                            <label class="label"><span class="label-text font-bold">SSH Public Key</span></label>
                            <textarea name="sshKey" required rows="4" class="textarea textarea-bordered textarea-primary font-mono" placeholder="ssh-ed25519 AAAAC3NzaC1..."></textarea>
                            <label class="label"><span class="label-text-alt">Paste your public key for edge server access.</span></label>
                        </div>

                        <button type="submit" class="btn btn-primary btn-block btn-lg mt-4" disabled={isLoading}>
                            {#if isLoading} <span class="loading loading-spinner"></span> {/if}
                            Activate My Account
                        </button>
                    </form>
                </div>
            </div>
        {/if}
        
    {/if}
</div>