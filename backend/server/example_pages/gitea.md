# Git & Gitea

Git tracks changes to your projects.

Gitea provides a private web interface where your Git repositories are stored and managed.

Your Gitea account uses:

- **Username:** `{{.Username}}`
- **Password:** the password created during onboarding
- **Gitea:** [{{.GiteaURL}}]({{.GiteaURL}})

---

## 1. Configure Git

Run this once on the server:

```bash
git config --global user.name "{{.Username}}"
git config --global user.email "{{.Email}}"
```
----------

## 2. Create a Project Repository

For an existing project, enter the project directory:
```bash
cd my_python_project
```
Initialize Git:
```bash
git init
```
----------

## 3. Create a `.gitignore`

Before committing anything, create a `.gitignore` file.

For a typical Python project:
```.gitignore
.venv/
__pycache__/
*.csv
*.dat
.env
```
Do not commit:
-   Virtual environments
-   Large datasets
-   Secrets or passwords
-   Generated files
-   Temporary files

----------

## 4. Create Your First Commit

Stage your files:
```bash
git add .
Create a commit:
git commit -m "Initial project backup"
```
A commit is a snapshot of your project at a particular point in time.

----------

## 5. Create the Gitea Repository

Open [{{.GiteaURL}}]({{.GiteaURL}}).

Create a new repository using **+ New Repository**.

For an existing local project, leave the repository empty.

Do not initialize it with a README or other files.

----------

## 6. Connect Your Project to Gitea

Replace `my_python_project` with your repository name:
```bash
git remote add origin {{.GiteaURL}}/{{.Username}}/my_python_project.git
git branch -M main
```
Push the project:
```bash
git push -u origin main
```
Use your Gitea username and password if Git asks for credentials.

----------

## 7. Daily Workflow

Once your repository is configured, your normal workflow is:
```bash
git add .
git commit -m "Describe what changed"
git push
```
For example:
```bash
git add .
git commit -m "Add data preprocessing pipeline"
git push
```
### Commit messages

Use short messages that describe the change.

Good:
```text
Add data preprocessing pipeline
Fix divide-by-zero error
Update analysis notebook
```
Avoid:
```text
update
stuff
changes
fixed
```

----------

## Jupyter Notebooks

Before committing a notebook, clear its generated outputs when appropriate.

Large embedded images, tables, and other outputs can make repositories unnecessarily large.

If you primarily work through Jupyter Lab, see the [Jupyter Lab guide](?docs=jupyter-lab&username={{.Username}}&email={{.Email}}{{range .Servers}}&server={{.Name}}{{end}}).

---

### To learn more about `git`
<a href="https://youtu.be/-iWaarLI7zI"><img src="https://img.youtube.com/vi/-iWaarLI7zI/0.jpg" alt="Git Basic concepts"></a>
