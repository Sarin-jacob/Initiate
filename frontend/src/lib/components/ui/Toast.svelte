<script>
    import { toasts, removeToast } from '../../stores/toast.js';
    import { fly, fade } from 'svelte/transition';
</script>

<div class="toast toast-top toast-center z-100 p-4 md:p-6">
    {#each $toasts as toast (toast.id)}
        <div 
            in:fly={{ y: 50, duration: 300 }} 
            out:fade={{ duration: 200 }}
            class="alert shadow-xl font-bold flex flex-row items-center justify-between gap-4 border-l-4
            {toast.type === 'error' ? 'alert-error border-error-content' : 'alert-success border-success-content'}"
        >
            <div class="flex items-center gap-3">
                {#if toast.type === 'error'}
                    <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                {:else}
                    <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                {/if}
                <span>{toast.message}</span>
            </div>
            <button class="btn btn-sm btn-ghost btn-circle opacity-70 hover:opacity-100" on:click={() => removeToast(toast.id)}>✕</button>
        </div>
    {/each}
</div>