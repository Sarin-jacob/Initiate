Hello {{.Username}},

Your development environment has been fully provisioned and is ready to use.

You now have access to:

- A dedicated Linux development environment in the following server(s)
    {{range .Servers}}1. _**{{.Name}}**_ : _`{{.Address}}`_
    {{end}}
- Private Git repositories through Gitea
- Jupyter Lab for interactive development

### Get Started

[Open Gitea]({{.GiteaURL}})

For everything else, use the guides below:

- [SSH & Server Access]({{.SystemURL}}/?docs=server-access&username={{.Username}}&email={{.Email}}{{range .Servers}}&server={{.Name}}{{end}})
- [Git & Gitea]({{.SystemURL}}/?docs=git-gitea&username={{.Username}}&email={{.Email}}{{range .Servers}}&server={{.Name}}{{end}})
- [Jupyter Lab]({{.SystemURL}}/?docs=jupyter-lab&username={{.Username}}&email={{.Email}}{{range .Servers}}&server={{.Name}}{{end}})

We recommend starting with **SSH & Server Access**.

Your Gitea username is `{{.Username}}`, and your server is accessed using the SSH key configured during onboarding.

Happy coding!