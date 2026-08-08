<script>
    export let username = "";
    export let giteaUrl = "";
    export let size = "10"; // default size in Tailwind sizing (w-10 h-10)

    // Extract first letter for the fallback
    $: initial = username ? username.charAt(0).toUpperCase() : "?";
    
    // Gitea provides a public avatar endpoint by username
    $: avatarSrc = giteaUrl && username ? `${giteaUrl}/user/avatar/${username}/-1` : null;
    
    let imageError = false;
</script>

<div class="avatar avatar-placeholder">
    {#if avatarSrc && !imageError}
        <div class="bg-base-200 rounded-full w-{size} h-{size}">
            <img src={avatarSrc} alt={username} on:error={() => imageError = true} />
        </div>
    {:else}
        <div class="bg-primary text-primary-content rounded-full w-{size} h-{size}">
            <span class="font-bold text-lg">{initial}</span>
        </div>
    {/if}
</div>