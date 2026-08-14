# Jupyter Lab

Jupyter Lab provides a browser-based development environment for Python, notebooks, terminals, and Git.

## 1. Start Jupyter Lab

From your personal computer, open a terminal and run:

{{range .Servers}}
### {{.Name}}

```bash
ssh {{$.Username}}@{{.Address}} -t "uv run jupyter lab"
```
{{end}}
---
The terminal will display a URL similar to:
```bash
https://0.0.0.0:38491/lab
```
Replace `0.0.0.0` with the server's IP address before opening the URL in your browser.

### Certificate Warning

Your browser may warn that the connection is not private.

This happens because the Jupyter server uses a self-signed certificate.

If you are connecting to your assigned server, use the browser option to proceed to the site.

Log in using the password configured for your environment.

----------

## 2. Working With Projects

Your Python projects are managed using `uv`.

For example:
```bash
uv add numpy pandas matplotlib
or 
!uv add numpy pandas matplotlib #inside jupyter cell to add libraries
```
Run Python programs with:
```bash
uv run python main.py
```
----------

## 3. Git in Jupyter Lab

Jupyter Lab includes a Git interface that allows you to perform common Git operations without using the command line.

You can use it to:
-   View changed files
-   Stage changes
-   Commit changes
-   Pull changes
-   Push changes

![Jupyter Git Extension](https://raw.githubusercontent.com/jupyterlab/jupyterlab-git/main/docs/figs/preview.gif)

For complete Git setup instructions, see the [Git & Gitea guide](?docs=git-gitea&username={{.Username}}&email={{.Email}}{{range .Servers}}&server={{.Name}}{{end}}).

----------

## 4. Starting a New Project

The recommended workflow is:

1.  Create a repository in Gitea.
2.  Clone the repository into Jupyter Lab.
3.  Work inside the cloned project directory.
4.  Commit and push your changes regularly.

Use the Git sidebar's **Clone a Repository** option and provide your Gitea repository URL.

----------

## 5. Existing Projects

If you already have a project directory on the server:
```bash
cd my_python_project
git init
```
Then follow the [Git & Gitea guide](?docs=git-gitea&username={{.Username}}&email={{.Email}}{{range .Servers}}&server={{.Name}}{{end}}) to connect it to Gitea.

----------

## 6. Keep Repositories Clean

Do not commit:
```.gitignore
.venv/
__pycache__/
*.csv
*.dat
.env
```
For Jupyter notebooks, **clear unnecessary notebook outputs** before committing.

---
