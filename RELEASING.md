# Releasing

## Tag and push a new release

Tag the version and push:

```
git tag -a v0.0.1 -m "Initial release"
git push origin v0.0.1
```

Goreleaser will pick up the new tag, and release it to Github.

## Commit messages

Pact uses the [Conventional Changelog](https://github.com/bcoe/conventional-changelog-standard/blob/master/convention.md)
commit message conventions. Please ensure you follow the guidelines, as they
help us automate our release process.

You can take a look at the git history (`git log`) to get the gist of it.
If you have questions, feel free to reach out in `#pact-js` in our [slack
community](https://pact-foundation.slack.com/).

If you'd like to get some CLI assistance, getting setup is easy:

```shell
npm install commitizen -g
npm i -g cz-conventional-changelog
```

`git cz` to commit and commitizen will guide you.