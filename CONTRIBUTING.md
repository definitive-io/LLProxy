# Contribution Guidelines

Thank you for your interest in contributing to this project!  We believe in the value of open source and that we can build better software as a community.


If you like the project, but just don't have time to contribute, that's fine too. There are other easy ways to support the project and show your appreciation, which we would also be very happy about:
 - Star the project
 - Follow us on [Twitter](https://twitter.com/definitiveio), or tweet about the project
 - Join our [Discord](https://discord.gg/CPJJfq87Vx) community chat
 - Refer this project in your own project's readme
 - Mention the project at local meetups and tell your friends/colleagues


> ### Legal Notice 
> When contributing to this project, you must agree that you have authored 100% of the content and that you have the necessary rights to the content.  All submitted code must be [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0) licensed to be incorporated into this project.


# Contributing

## The Basics
- See our [SECURITY.md](SECURITY.md) if you are submitting a security issue
- Make sure you have a [GitHub](https://github.com) account.
- Submit an [Issue](https://github.com/definitive-io/LLProxy/issues) if one does not already exist.
    - Clearly describe the issue including steps to reproduce when it is a bug.
    - Make sure you fill in the earliest version that you know has the issue.
    - A ticket is not necessary for trivial changes
- Fork the repository on GitHub.
- Author your change
- Submit a Pull Request

## Conduct
- Be considerate to the rest of our community by adhering to our [community code of conduct](CONDUCT.md)

## Setting up your environment
- Install [Go 1.20](https://go.dev/doc/install)
- (Recommended) Install [pre-commit](https://pre-commit.com/)

## Making Changes
- Create a topic branch from where you want to base your work
  - This is usually off the main branch
  - To quickly create a topic branch based on main, run 
    ```sh
    git checkout -b fix/main/my_contribution main
    ```

- Make commits of logical and atomic units.

- Make sure you have added the necessary tests for your changes.

- Make sure that you have run `go fmt` before submitting your PR.  This will be done automatically if you have setup [pre-commit](https://pre-commit.com/) correctly.

## Commit Messages

- Good commit messages serve at least three important purposes:
  - To speed up the reviewing process.
  - To help us write a good release note.
  - To help the future maintainers to find out why a particular change was made.
  - Start the line with "Fix", "Add", "Change" instead of "Fixed", "Added", "Changed"

- Structure your commit message like this:

    > Short (50 chars or less) summary of changes
    > 
    > More detailed explanatory text, if necessary.  Wrap it to about 72
    > characters or so. 
    >
    > Further paragraphs come after blank lines.
    > 
    > - Bullet points are okay, too

## File Headers
- Don't forget your [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0) license headers on new files.

## Code Style
- Please adhere to the [existing coding style](https://google.github.io/styleguide/go/) for consistency.


## Submit your Pull Request
- Clearly describe your change in the PR message
- Link to the [Issue](https://github.com/definitive-io/LLProxy/issues) it is addressing
