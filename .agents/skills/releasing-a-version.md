---
name: releasing-a-version
description: Use when the user asks to cut, tag, or publish a new release of gh-markdown-preview. Covers version bump, tag, push, GitHub release, and the launcher-script tag update that must accompany it.
---

# Releasing a version of gh-markdown-preview

This repo has a quirk: the `gh-markdown-preview` launcher script in the repo root has a hardcoded `tag=...` line that determines which release binary the `gh` extension downloads at runtime. **Forgetting to bump this means installed users keep getting the old binary even after a new release.**

## Steps

1. **Pick the version.** Bump from the latest tag (`git tag --sort=-v:refname | head -1`). Use semver: features → minor, fixes → patch.

2. **Verify the tree is clean and tests pass.**
   ```
   git status
   go build ./... && go vet ./... && go test ./...
   ```

3. **Update the launcher script.** Edit `gh-markdown-preview` (the bash script at repo root) and change the `tag="vX.Y.Z"` line to the new version. Commit this with the feature commit, or as its own commit before tagging — but it MUST point at the tag you're about to create.

4. **Push `master`.**
   ```
   git push origin master
   ```

5. **Tag and push the tag.**
   ```
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

6. **Create the GitHub release.** This repo lives under the `ofirgall` GitHub account, but the default active `gh` account on this machine is `ofir-drift`. **ALWAYS run `gh auth switch --user ofirgall` before any `gh` command in this repo**, even if you switched earlier in the session — the active account can flip back between commands. Don't assume; switch every time.
   ```
   gh auth switch --user ofirgall
   gh release create vX.Y.Z --title "vX.Y.Z" --notes "..."
   ```

   Release notes: short bullet list of user-visible changes. No "internal refactor" noise.

7. **Verify.** The release page should show at `https://github.com/ofirgall/gh-markdown-preview/releases/tag/vX.Y.Z`.

## Notes

- The launcher script always builds from source via `go build` — this fork does not publish prebuilt release binaries. The `tag=` line in the script controls which `dist/<tag>/` directory caches the built binary; bumping it forces a rebuild on next invocation.
- There is no CI release workflow in this repo — releases are manual. No prebuilt assets are uploaded.
- The remote is `git@github.com:ofirgall/gh-markdown-preview.git` (a fork of `yusukebe/gh-markdown-preview`). Do not push tags upstream.
