<script>
    export let username = "";
    export let giteaUrl = "";
    export let sizeClass = "w-10 h-10"; 

    $: initial = username ? username.charAt(0).toUpperCase() : "?";
    $: avatarSrc = giteaUrl && username ? `${giteaUrl}/user/avatar/${username}/-1` : null;
    
    let imageError = false;
</script>

<div class="avatar avatar-placeholder">
    {#if avatarSrc && !imageError}
        <div class="bg-base-200 rounded-full {sizeClass}">
            <img src={avatarSrc} alt={username} on:error={() => imageError = true} />
        </div>
    {:else}
        <div class="bg-primary text-primary-content rounded-full {sizeClass}">
            <span class="font-bold text-lg">{initial}</span>
        </div>
    {/if}
</div>