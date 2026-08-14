# SSH Keys

Server access uses SSH keys instead of passwords.

An SSH key consists of two files:

- **Private key** — stays on your computer and must never be shared.
- **Public key** — uploaded during account setup and stored on the server.

Your computer proves its identity by using the private key that matches the public key registered on your account.

## Generate an SSH Key

If you don't already have an Ed25519 key, open a terminal on your computer and run:
```bash
ssh-keygen -t ed25519 -C "{{.Email}}"
```
Press **Enter** to use the default file location.

When prompted for a passphrase, using one is strongly recommended.

Find Your Public Key
--------------------

### macOS / Linux / WSL
```bash
cat ~/.ssh/id_ed25519.pub
```
### Windows PowerShell
```powershell
Get-Content ~/.ssh/id_ed25519.pub
```
Your public key will look similar to:
```text
ssh-ed25519 AAAAC3... user@example.com
```
Copy the **entire line** and provide it during account setup.

Important
---------
Never share:
```bash
~/.ssh/id_ed25519
```
That is your **private key**.
Only the following file should be shared:
```bash
~/.ssh/id_ed25519.pub
```