<script>
    import { onMount } from 'svelte';
    export let slug = "";

    let title = "";
    let htmlContent = "";
    let fetchError = "";
    let isLoading = true;

    onMount(async () => {
        try {
            // We use the markdown preview endpoint to convert the raw content to HTML safely
            const renderRes = await fetch(`/api/docs/${slug}`);
            if (!renderRes.ok) throw new Error("Document not found");

            const renderData = await renderRes.json();
            title = renderData.title;
            htmlContent = renderData.html_content;

        } catch (err) {
            fetchError = err.message;
        } finally {
            isLoading = false;
        }
    });
</script>

<div class="min-h-screen bg-base-200 p-4 md:p-8 flex justify-center items-start">
    <div class="card bg-base-100 shadow-xl max-w-4xl w-full">
        {#if isLoading}
            <div class="p-12 flex justify-center"><span class="loading loading-spinner loading-lg text-primary"></span></div>
        {:else if fetchError}
            <div class="p-12 text-center text-error font-bold text-xl">{fetchError}</div>
        {:else}
            <div class="bg-primary text-primary-content p-8 text-center rounded-t-2xl">
                <h1 class="text-3xl font-bold">{title}</h1>
            </div>
            <div class="p-8 md:p-12 prose prose-sm md:prose-base max-w-none">
                {@html htmlContent}
            </div>
        {/if}
    </div>
</div>