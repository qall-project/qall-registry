# Contribute to Qall Registry

Qall Registry is an open-source component of Qall, developed by Scaleway's R&D teams and released under the Apache 2.0 license.

The Registry provides storage and distribution for Qall computation objects, allowing workflows and reusable computation components to be shared and retrieved independently from their execution environment.

Contributions are welcome through GitHub, whether you want to report a bug, improve documentation, suggest an enhancement, or contribute code.

## Topics

- [Reporting Security Issues](#reporting-security-issues)
- [Reporting Issues](#reporting-issues)
- [Suggesting a Feature](#suggesting-a-feature)
- [Contributing Code](#contributing-code)
- [Pull Request Guidelines](#pull-request-guidelines)
- [Development Principles](#development-principles)
- [Community Guidelines](#community-guidelines)

## Reporting Security Issues

At Scaleway, we take security seriously.

If you find a security issue in Qall Registry, please notify us by sending an email to [security@scaleway.com](mailto:security@scaleway.com).

Please **do not create a GitHub issue** for security vulnerabilities.

We will follow up with you promptly with more information and a plan for remediation.

We currently do not offer a paid security bounty program, but we greatly appreciate your help in making Qall and its surrounding ecosystem more secure.

## Reporting Issues

Bug reports and detailed issue reports are valuable contributions to Qall Registry.

Before opening a new issue, please check the existing issues to see whether a similar report already exists. If it does, add a 👍 reaction and provide any additional information that may help us investigate.

When reporting an issue, please include as much relevant information as possible, such as:

- Qall Registry version or commit
- Operating system
- How the Registry was deployed or accessed
- Relevant client or server configuration
- Steps to reproduce the issue
- Expected and actual behavior
- Relevant logs or error messages

For issues involving the Registry API or client/server interaction, please also provide the relevant API or protocol version when available.

## Suggesting a Feature

We welcome ideas that improve the storage, distribution and reuse of Qall computation objects.

When suggesting a feature, please consider:

- **What problem does it solve?**
- **Who benefits from it?**
- **What would the expected API or user experience look like?**
- **Does it preserve the separation between computation objects and their execution environment?**
- **Does it fit with Qall's content-addressed model?**
- **How does it affect compatibility between clients and Registry servers?**

For larger changes, opening an issue first is recommended so the design can be discussed before implementation.

## Contributing Code

### Submit Code

To contribute code:

1. Fork the project.
2. Create a topic branch from the current `main` branch.
3. Make your changes.
4. Add or update tests covering the changes.
5. Update the API or documentation when relevant.
6. Run the relevant checks locally.
7. Push your commits to your fork.
8. Open a pull request against the `main` branch.

Keep changes focused and avoid combining unrelated changes in the same pull request.

For changes affecting the Registry API, storage model or interoperability, we recommend discussing the approach in an issue before implementation.

### Pull Request Guidelines

The goal of the pull request process is to make changes easy to review, understand and maintain.

Please:

- **Use a clear pull request title** describing what is being changed.
- **Keep pull requests focused** on a specific change or problem.
- **Include tests** for new or modified behavior.
- **Update API documentation** when changing public interfaces.
- **Keep the implementation readable** and avoid unnecessary complexity.
- **Explain design decisions** when they are not obvious from the code.
- **Mark work-in-progress pull requests** with `[WIP]` when they are not ready for review.
- **Keep the pull request up to date** with the current `main` branch.

If you are addressing an existing issue, reference it from the pull request description.

Please do not merge `main` into your topic branch. Rebase your branch when necessary to keep it up to date.

Maintainers may request changes, additional tests or documentation before merging.

## Development Principles

Qall Registry is designed around a few principles that are particularly relevant when contributing:

- **Content-addressed**: computation objects should be identified by their content rather than by infrastructure-specific identifiers.
- **Immutable computation objects**: once published, an object should remain reproducible and addressable by its content.
- **Provider and infrastructure agnostic**: the Registry should not depend on a specific execution provider or infrastructure.
- **Separation of concerns**: the Registry is responsible for storing and distributing computation objects, not for executing them.
- **Interoperable interfaces**: client and server interfaces should remain clearly defined and suitable for independent implementations.
- **Composable computations**: stored computation objects should be usable as building blocks for larger Qall workflows.
- **Reproducibility**: changes to a computation should result in a distinct object rather than silently modifying an existing one.

When adding a new capability, consider whether it belongs in the Registry itself or should instead be implemented in the Qall SDK, Qall Daemon, or another execution component.

## Client and Server Changes

Qall Registry contains both client and server components.

When changing a public interface:

- Keep client and server behavior consistent
- Update the relevant protocol or API definitions
- Add tests covering both sides when appropriate
- Consider compatibility with existing Registry clients and servers
- Avoid introducing execution-specific logic into the Registry API

Changes to the Registry protocol should be treated as API changes and reviewed accordingly.

## Community Guidelines

Please be respectful and constructive when participating in the Qall community.

Questions, discussions, ideas and constructive criticism are welcome. When proposing changes, focus on the problem being solved and help maintain a collaborative environment.

Thank you for contributing to **Qall Registry**!