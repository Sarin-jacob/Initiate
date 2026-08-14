## Welcome, {{.Username}}!

Your account is ready to be activated.

Complete the setup below to configure your permanent access.

### 1. Set Up Your SSH Key

Server access uses SSH keys instead of passwords.

If you already have an Ed25519 SSH key, you can use your existing public key.

Otherwise, follow the guide below to generate one:

<a href="{{.SystemURL}}/api/invite/{{.Token}}/page/ssh-keys" class="btn btn-sm rounded-full bg-base-200 border-base-300 text-base-content font-normal hover:bg-base-300 shadow-none no-underline">
  SSH Key Setup
</a>

### 2. Complete Account Setup

Enter your SSH public key and create your account credentials using the onboarding form.

> **Important:** Keep your SSH private key on your own computer. Never share it with anyone.

### What's Next?

Once your account has been activated, you'll receive a welcome email with links to the guides for:

- Connecting to your server
- Using Python and `uv`
- Working with Git and Gitea
- Running Jupyter Lab

Complete the setup above to get started.