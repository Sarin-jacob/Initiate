# Server Access & Python

This guide covers connecting to your Linux development environment and managing Python projects.

## 1. Connect to Your Server

Open a terminal on your computer and connect using SSH:

{{range .Servers}}
### {{.Name}}

```bash
ssh {{$.Username}}@{{.Address}}
```
{{end}}

If you have problems connecting, make sure that:

1.  Your SSH public key was added during onboarding.
2.  Your private key is available on your computer.
3.  You are using the correct username and server address.

----------

## 2. Python Environment

Your home directory is pre-configured with [`uv`](https://docs.astral.sh/uv/), a fast Python package and project manager.

Python projects should use `uv` rather than installing packages globally.

### Install dependencies

For example:
```bash
uv add numpy pandas matplotlib
```
This updates your project's `pyproject.toml` and keeps dependencies isolated.

### Run a Python program
```bash
uv run python main.py
```
### Run Jupyter Lab
```bash
uv run jupyter lab
```
See the [Jupyter Lab guide](?docs=jupyter-lab&username={{.Username}}&email={{.Email}}{{range .Servers}}&server={{.Name}}{{end}}) for the recommended way to launch it remotely.

----------

## Important Rules

### Do not delete `pyproject.toml`

The `pyproject.toml` file defines your Python project and its dependencies.

### Do not use `pip`

Avoid:
```bash
pip install numpy
```
Use:
```bash
uv add numpy
```
This keeps your environment reproducible and prevents dependency conflicts.

### Do not install packages globally

Keep project dependencies inside the project environment managed by `uv`.
