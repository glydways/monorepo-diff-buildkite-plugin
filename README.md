[![e2e status](https://badge.buildkite.com/719d0b895285367c9c57a09e07f1e853148d2509f0667e0ae8.svg?branch=master)](https://buildkite.com/glydways/monorepo-diff-buildkite-plugin)
[![codecov](https://codecov.io/gh/glydways/monorepo-diff-buildkite-plugin/branch/master/graph/badge.svg?token=DQ3B4FIYD2)](https://codecov.io/gh/glydways/monorepo-diff-buildkite-plugin)
[![Publish](https://github.com/glydways/monorepo-diff-buildkite-plugin/actions/workflows/publish.yml/badge.svg)](https://github.com/glydways/monorepo-diff-buildkite-plugin/actions/workflows/publish.yml)

# monorepo-diff-buildkite-plugin

NOTE - This is a fork of the original `monorepo-diff-buildkite-plugin` to be able to use additional plugins in downstream jobs.
This should not be needed once upstream [PR merges](https://github.com/monebag/monorepo-diff-buildkite-plugin/pull/141).

This plugin will assist you in triggering pipelines by watching folders in your `monorepo`.

Check out this post to learn [**How to set up Continuous Integration for monorepo using Buildkite**](https://adikari.medium.com/set-up-continuous-integration-for-monorepo-using-buildkite-61539bb0ed76).

## Using the plugin

If the version number is not provided then the most recent version of the plugin will be used. Do not use version number as `master` or any branch names.

### Simple

```yaml
steps:
  - label: "Triggering pipelines"
    plugins:
      - glydways/monorepo-diff#v2.6.7:
          diff: "git diff --name-only HEAD~1"
          watch:
            - path: "bar-service/"
              config:
                command: "echo deploy-bar"
            - path: "foo-service/"
              config:
                trigger: "deploy-foo-service"
```

### Detailed

```yaml
steps:
  - label: "Triggering pipelines"
    plugins:
      - glydways/monorepo-diff#v2.6.7:
          diff: "git diff --name-only $(head -n 1 last_successful_build)"
          interpolation: false
          env:
            - env1=env-1 # this will be appended to all env configuration
          hooks:
            - command: "echo $(git rev-parse HEAD) > last_successful_build"
          notify:
            - email: foo@gmail.com
            - basecamp_campfire: https://basecamp-url
            - webhook: https://webhook-url
            - pagerduty_change_event: 636d22Yourc0418Key3b49eee3e8
            - github_commit_status:
                context: my-custom-status
            - slack: '@someuser'
              if: build.state === "passed"
          watch:
            - path:
                - "ops/terraform/"
                - "ops/templates/terraform/"
              config:
                command: "buildkite-agent pipeline upload ops/.buildkite/pipeline.yml"
                label: "Upload pipeline"
                # following configs are available in command. notify is not available in trigger step
                notify:
                  - basecamp_campfire: https://basecamp-url
                  - github_commit_status:
                      context: my-custom-status
                  - slack: '@someuser'
                    if: build.state === "passed"
                # soft_fail: true
                soft_fail:
                  - exit_status: 1
                  - exit_status: "255"
                retry:
                  automatic:
                  - limit: 2
                    exit_status: -1
                concurrency: 1
                concurrency_group: "ops/terraform"
                agents:
                  queue: performance
                artifacts:
                  - "logs/*"
                env:
                  - FOO=bar
                plugins:
                  - seek-oss/aws-sm#v2.3.1:
                    env:
                      AUTH_SECRET:
                        secret-id: "secret/id"
                        json-key: ".key"
            - path: "foo-service/"
              config:
                trigger: "deploy-foo-service"
                label: "Triggered deploy"
                if: build.pull_request.base_branch == "main"
                branches:
                  - main
                  - "release/*"
                build:
                  message: "Deploying foo service"
                  meta_data:
                    build_number: "123"
                  env:
                    - HELLO=123
                    - AWS_REGION

          wait: true
```

## Configuration

## `diff` (optional)

This will run the script provided to determine the folder changes.
Depending on your use case, you may want to determine the point where the branch occurs
https://stackoverflow.com/questions/1527234/finding-a-branch-point-with-git and perform a diff against the branch point.

#### Sample output:
```
README.md
lib/trigger.bash
tests/trigger.bats
```

Default: `git diff --name-only HEAD~1`

#### Examples:

`diff: ./diff-against-last-successful-build.sh`

```bash
#!/bin/bash

set -ueo pipefail

LAST_SUCCESSFUL_BUILD_COMMIT="$(aws s3 cp "${S3_LAST_SUCCESSFUL_BUILD_COMMIT_PATH}" - | head -n 1)"
git diff --name-only "$LAST_SUCCESSFUL_BUILD_COMMIT"
```

`diff: ./diff-against-last-built-tag.sh`

```bash
#!/bin/bash

set -ueo pipefail

LATEST_BUILT_TAG=$(git describe --tags --match foo-service-* --abbrev=0)
git diff --name-only "$LATEST_TAG"
```

## `interpolation` (optional)

This controls the pipeline interpolation on upload, and defaults to `true`.
If set to `false` it adds `--no-interpolation` to the `buildkite pipeline upload`,
to avoid trying to interpolate the commit message, which can cause failures.

## `env` (optional)

The object values provided in this configuration will be appended to `env` property of all steps or commands.

## `MONOREPO_DIFF_TRIGGER_ALL` (env var)

When set to `"true"` in the build environment, the plugin skips the `diff` step
and emits one trigger/command step per entry in `watch:`. Useful for scheduled
fan-out builds (e.g. an hourly health check that exercises every downstream
pipeline regardless of changed paths).

```yaml
# Schedule this build with env: MONOREPO_DIFF_TRIGGER_ALL=true
steps:
  - label: ":sparkles: trigger every pipeline"
    plugins:
      - glydways/monorepo-diff#v2.6.7:
          watch:
            - path: "foo-service/"
              config:
                trigger: "deploy-foo"
            - path: "bar-service/"
              config:
                trigger: "deploy-bar"
```

## `log_level` (optional)

Add `log_level` property to set the log level. Supported log levels are `debug` and `info`. Defaults to `info`.

```yaml
steps:
  - label: "Triggering pipelines"
    plugins:
      - glydways/monorepo-diff#v2.6.7:
          diff: "git diff --name-only HEAD~1"
          log_level: "debug" # defaults to "info"
          watch:
            - path: "foo-service/"
              config:
                trigger: "deploy-foo-service"
```

## `watch`

Declare a list of

```yaml
- path: app/cms/
  config: # Required [trigger step configuration]
    trigger: cms-deploy # Required [trigger pipeline slug]
- path:
    - services/email
    - assets/images/email
  config:
    trigger: email-deploy
```

### `path`

If the `path` specified here in the appears in the `diff` output, a `trigger` step will be added to the dynamically generated pipeline.yaml

A list of paths can be provided to trigger the desired pipeline. Changes in any of the paths will initiate the pipeline provided in trigger.

A `path` can also be a glob pattern. For example specify `path: "**/*.md"` to match all markdown files.

### `config`

Configuration supports 2 different step types.

- [Trigger](https://buildkite.com/docs/pipelines/trigger-step)
- [Command](https://buildkite.com/docs/pipelines/command-step)

#### Trigger

The configuration for the `trigger` step https://buildkite.com/docs/pipelines/trigger-step

By default, it will pass the following values to the `build` attributes unless an alternative values are provided

```yaml
- path: app/cms/
  config:
    trigger: cms-deploy
    build:
      commit: $BUILDKITE_COMMIT
      branch: $BUILDKITE_BRANCH
```

Note that only `message`, `branch` and `commit` are carried over. Nothing else about
the current build — pull request number, base branch, source — reaches the triggered
build, because Buildkite creates it through the API with no pull request association.
Pass anything else you need explicitly via `build.env` or `build.meta_data`:

```yaml
- path: app/cms/
  config:
    trigger: cms-deploy
    build:
      env:
        # A bare key takes its value from the current build's environment.
        - BUILDKITE_PULL_REQUEST
        - BUILDKITE_PULL_REQUEST_BASE_BRANCH
        - BUILDKITE_PULL_REQUEST_REPO
      meta_data:
        release_channel: "stable"
```

`build.env` is a list of `KEY=value` strings (or bare `KEY` to inherit), while
`build.meta_data` is a map. Note that these env vars are visible to the triggered
build's scripts but do not give it a real pull request association, so downstream
`if:` expressions on `build.pull_request.*` still will not resolve.

#### `if` and `branches` (optional)

Both step types accept the standard Buildkite conditionals, letting you gate an
individual watch entry on top of its path match.

```yaml
- path: app/cms/
  config:
    trigger: cms-deploy
    # Only trigger when the pull request targets main.
    if: build.pull_request.base_branch == "main"
- path: app/api/
  config:
    trigger: api-deploy
    # `branches` accepts a single branch or a list, and matches the *current*
    # branch (BUILDKITE_BRANCH) — not the pull request base branch.
    branches:
      - main
      - "release/*"
```

See [conditionals](https://buildkite.com/docs/pipelines/conditionals) and
[branch configuration](https://buildkite.com/docs/pipelines/branch-configuration)
for the supported syntax.

These are evaluated by Buildkite on the *generated* step, in the build running the
plugin, so `build.pull_request.base_branch` still resolves here. Put the condition
on the watch entry rather than in the triggered pipeline — once the downstream build
starts, its pull request context is gone.

To gate every watch entry at once, put the condition on the step that runs the
plugin instead:

```yaml
steps:
  - label: "Triggering pipelines"
    if: build.pull_request.base_branch == "main"
    plugins:
      - glydways/monorepo-diff#v2.6.7:
          watch:
            - path: "app/cms/"
              config:
                trigger: "cms-deploy"
```

### `wait` (optional)

Default: `true`

By setting `wait` to `true`, the build will wait until the triggered pipeline builds are successful before proceeding

### `hooks` (optional)

Currently supports a list of `commands` you wish to execute after the `watched` pipelines have been triggered

```yaml
hooks:
  - command: upload unit tests reports
  - command: echo success
```

#### Command

```yaml
steps:
  - label: "Triggering pipelines"
    plugins:
      - glydways/monorepo-diff#v2.6.7:
          diff: "git diff --name-only HEAD~1"
          watch:
            - path: app/cms/
              config:
                group: ":rocket: deployment"
                command: "netlify --production deploy"
                label: ":netlify: Deploy to production"
                agents:
                  queue: "deploy"
                env:
                  - FOO=bar
```

There is currently limited support for command configuration. Only the `command` property can be provided at this point in time.

Using commands, it is also possible to use this to upload other pipeline definitions

```yaml
- path: frontend/
  config:
    command: "buildkite-agent pipeline upload ./frontend/.buildkite/pipeline.yaml"
- path: infrastructure/
  config:
    command: "buildkite-agent pipeline upload ./infrastructure/.buildkite/pipeline.yaml"
- path: backend/
  config:
    command: "buildkite-agent pipeline upload ./backend/.buildkite/pipeline.yaml"
```

## How to Contribute

Please read [contributing guide](https://github.com/glydways/monorepo-diff-buildkite-plugin/blob/master/CONTRIBUTING.md).
