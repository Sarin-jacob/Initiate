<script>
    import { createEventDispatcher } from 'svelte';
    const dispatch = createEventDispatcher();
    
    let password = '';
    let isLoggingIn = false;
    let errorMsg = '';

    async function handleLogin(e) {
        e.preventDefault();
        isLoggingIn = true;
        errorMsg = '';

        try {
            const res = await fetch('/api/admin/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ password })
            });
            const contentType = res.headers.get('content-type') || '';
            const data = contentType.includes('application/json') ? await res.json() : await res.text();
            if (!res.ok) { throw new Error( typeof data === 'object' ? data.message || 'Invalid password' : data || 'Invalid password' ); }
            
            localStorage.setItem('nexus_jwt', data.token);
            dispatch('success');
        } catch (err) {
            errorMsg = err.message;
        } finally {
            isLoggingIn = false;
        }
    }
</script>

<div class="min-h-screen bg-base-200 flex items-center justify-center p-4">
    <div class="card bg-base-100 shadow-2xl border border-base-300 w-full max-w-sm">
        <div class="card-body">
            <div class="flex items-center justify-center mb-6">
                <div class="w-12 h-12 rounded-xl bg-primary text-primary-content flex items-center justify-center font-bold text-2xl shadow-sm">N</div>
            </div>
            <h2 class="card-title justify-center text-2xl mb-2">NexusIAM Admin</h2>
            
            {#if errorMsg}
                <div class="alert alert-error shadow-sm p-3 text-sm">{errorMsg}</div>
            {/if}

            <form on:submit={handleLogin} class="space-y-4 mt-4">
                <div class="form-control">
                    <input type="password" bind:value={password} required placeholder="Admin Password" class="input input-bordered w-full" autofocus />
                </div>
                <button type="submit" class="btn btn-primary w-full" disabled={isLoggingIn}>
                    {#if isLoggingIn} <span class="loading loading-spinner loading-sm"></span> {/if}
                    Secure Login
                </button>
            </form>
        </div>
    </div>
</div>