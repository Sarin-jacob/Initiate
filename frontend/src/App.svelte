<script>
    import AdminDashboard from './lib/AdminDashboard.svelte';
    import Onboarding from './lib/Onboarding.svelte';

    // Basic client-side routing based on URL parameters
    const urlParams = new URLSearchParams(window.location.search);
    const inviteToken = urlParams.get('token');
</script>

<main class="max-w-4xl mx-auto p-6 mt-10">
    {#if inviteToken}
        <!-- If there's a token in the URL, they are an invitee -->
        <Onboarding token={inviteToken} />
    {:else}
        <!-- Otherwise, assume Admin view (Secured by JWT later) -->
        <AdminDashboard />
    {/if}
</main>