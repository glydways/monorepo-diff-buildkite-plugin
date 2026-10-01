package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginWithEmptyParameter(t *testing.T) {
	_, err := initializePlugin("[]")

	assert.EqualError(t, err, "could not initialize plugin")
}

func TestPluginWithInvalidParameter(t *testing.T) {
	_, err := initializePlugin("invalid")

	assert.EqualError(t, err, "failed to parse plugin configuration")
}

func TestPluginShouldHaveDefaultValues(t *testing.T) {
	param := `[{
		"github.com/glydways/monorepo-diff-buildkite-plugin#commit": {}
	}]`

	got, _ := initializePlugin(param)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
	}

	assert.Equal(t, expected, got)
}

func TestPluginWithValidParameter(t *testing.T) {
	param := ""
	got, err := initializePlugin(param)
	expected := Plugin{}

	assert.EqualError(t, err, "failed to parse plugin configuration")
	assert.Equal(t, expected, got)
}

func TestPluginShouldUnmarshallCorrectly(t *testing.T) {
	param := `[{
		"github.com/glydways/monorepo-diff-buildkite-plugin#commit": {
			"diff": "cat ./hello.txt",
			"wait": true,
			"log_level": "debug",
			"interpolation": true,
			"hooks": [
				{ "command": "some-hook-command" },
				{ "command": "another-hook-command" }
			],
			"env": [
				"env1=env-1",
				"env2=env-2",
				"env3"
			],
		"notify": [
				{ "email": "foo@gmail.com" },
				{ "email": "bar@gmail.com" },
				{ "basecamp_campfire": "https://basecamp-url" },
				{ "webhook": "https://webhook-url", "if": "build.state === 'failed'" },
				{ "pagerduty_change_event": "636d22Yourc0418Key3b49eee3e8" },
				{ "github_commit_status": { "context" : "my-custom-status" } },
				{ "slack": "@someuser", "if": "build.state === 'passed'" }
			],
			"watch": [
				{
					"path": "watch-path-1",
					"config": {
						"trigger": "service-2",
						"build": {
							"message": "some message",
							"meta_data": {
								"foo": "bar"
							}
						}
					}
				},
				{
					"path": "watch-path-1",
					"config": {
						"command": "echo hello-world",
						"env": [
							"env4", "hi= bye"
						],
						"concurrency": 1,
						"concurrency_group": "payments/deploy",
						"concurrency_method": "eager",
						"soft_fail": [{
							"exit_status": "*"
						}],
						"notify": [
							{ "email": "foo@gmail.com" },
							{ "email": "bar@gmail.com" },
							{ "basecamp_campfire": "https://basecamp-url" },
							{ "webhook": "https://webhook-url", "if": "build.state === 'failed'" },
							{ "pagerduty_change_event": "636d22Yourc0418Key3b49eee3e8" },
							{ "github_commit_status": { "context" : "my-custom-status" } },
							{ "slack": "@someuser", "if": "build.state === 'passed'" }
						]
					}
				},
				{
					"path": [
						"watch-path-1",
						"watch-path-2"
					],
					"config": {
						"trigger": "service-1",
						"label": "hello",
						"if": "build.pull_request.base_branch == \"main\"",
						"branches": ["main", "release/*"],
						"build": {
							"message": "build message",
							"branch": "current branch",
							"commit": "commit-hash",
							"env": [
								"foo =bar",
								"bar= foo"
							]
						},
						"async": true,
						"agents": {
							"queue": "queue-1",
							"database": "postgres"
						},
						"artifacts": [ "artifiact-1" ],
						"soft_fail": [{
							"exit_status": 127
						}]
					}
				},
				{
					"path": "watch-path-1",
					"config": {
						"group": "my group",
						"command": "echo hello-group",
						"env": [
							"env4", "hi= bye"
						],
						"soft_fail": true
					}
				}
			]
		}
	}]`

	got, _ := initializePlugin(param)

	expected := Plugin{
		Diff:          "cat ./hello.txt",
		Wait:          true,
		LogLevel:      "debug",
		Interpolation: true,
		Hooks: []HookConfig{
			{Command: "some-hook-command"},
			{Command: "another-hook-command"},
		},
		Env: map[string]string{
			"env1": "env-1",
			"env2": "env-2",
			"env3": "env-3",
		},
		Notify: []PluginNotify{
			{Email: "foo@gmail.com"},
			{Email: "bar@gmail.com"},
			{Basecamp: "https://basecamp-url"},
			{Webhook: "https://webhook-url", Condition: "build.state === 'failed'"},
			{PagerDuty: "636d22Yourc0418Key3b49eee3e8"},
			{GithubStatus: GithubStatusNotification{Context: "my-custom-status"}},
			{Slack: "@someuser", Condition: "build.state === 'passed'"},
		},
		Watch: []WatchConfig{
			{
				Paths: []string{"watch-path-1"},
				Step: Step{
					Trigger: "service-2",
					Build: Build{
						Message: "some message",
						Branch:  "go-rewrite",
						Commit:  "123",
						Env: map[string]string{
							"env1": "env-1",
							"env2": "env-2",
							"env3": "env-3",
						},
						MetaData: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			{
				Paths: []string{"watch-path-1"},
				Step: Step{
					Command: "echo hello-world",
					Env: map[string]string{
						"env1": "env-1",
						"env2": "env-2",
						"env3": "env-3",
						"env4": "env-4",
						"hi":   "bye",
					},
					SoftFail:          []interface{}{map[string]interface{}{"exit_status": "*"}},
					Concurrency:       1,
					ConcurrencyGroup:  "payments/deploy",
					ConcurrencyMethod: "eager",
					Notify: []StepNotify{
						{Basecamp: "https://basecamp-url"},
						{GithubStatus: GithubStatusNotification{Context: "my-custom-status"}},
						{Slack: "@someuser", Condition: "build.state === 'passed'"},
					},
				},
			},
			{
				Paths: []string{"watch-path-1", "watch-path-2"},
				Step: Step{
					Trigger: "service-1",
					Label:   "hello",
					Build: Build{
						Message: "build message",
						Branch:  "current branch",
						Commit:  "commit-hash",
						Env: map[string]string{
							"foo":  "bar",
							"bar":  "foo",
							"env1": "env-1",
							"env2": "env-2",
							"env3": "env-3",
						},
					},
					Async:     true,
					Condition: `build.pull_request.base_branch == "main"`,
					Branches:  []interface{}{"main", "release/*"},
					Agents:    map[string]string{"queue": "queue-1", "database": "postgres"},
					Artifacts: []string{"artifiact-1"},
					SoftFail: []interface{}{map[string]interface{}{
						"exit_status": float64(127),
					}},
				},
			},
			{
				Paths: []string{"watch-path-1"},
				Step: Step{
					Group:   "my group",
					Command: "echo hello-group",
					Env: map[string]string{
						"env1": "env-1",
						"env2": "env-2",
						"env3": "env-3",
						"env4": "env-4",
						"hi":   "bye",
					},
					SoftFail: true,
				},
			},
		},
	}

	assert.Equal(t, expected, got)
}

func TestPluginShouldOnlyFullyUnmarshallItselfAndNotOtherPlugins(t *testing.T) {
	param := `[
		{
			"github.com/example/example-plugin#commit": {
				"env": {
					"EXAMPLE_TOKEN": {
						"json-key": ".TOKEN",
						"secret-id": "global/example/token"
					}
				}
			}
		},
		{
			"github.com/glydways/monorepo-diff-buildkite-plugin#commit": {
				"watch": [
					{
						"env": [
							"EXAMPLE_TOKEN"
						],
						"path": [
							".buildkite/**/*"
						],
						"config": {
							"label": "Example label",
							"command": "echo hello world\\n"
						}
					}
				]
			}
		}
	]
	`
	_, err := initializePlugin(param)
	assert.NoError(t, err)
}

func TestPluginShouldErrorIfPluginConfigIsInvalid(t *testing.T) {
	param := `[
		{
			"github.com/glydways/monorepo-diff-buildkite-plugin#commit": {
				"env": {
					"anInvalidKey": { "nested": "An Invalid Value" }
				},
				"watch": [
					{
						"path": [
							".buildkite/**/*"
						],
						"config": {
							"label": "Example label",
							"command": "echo hello world\\n"
						}
					}
				]
			}
		}
	]
	`
	_, err := initializePlugin(param)
	assert.Error(t, err)
}

func TestParseEnv(t *testing.T) {
	t.Setenv("FROM_AGENT", "agent-value")

	tests := []struct {
		name string
		raw  string
		want map[string]string
	}{
		{"nil", ``, nil},
		{"list", `["FOO=bar", " SPACED = value "]`, map[string]string{"FOO": "bar", "SPACED": "value"}},
		{"list keeps text after second equals", `["FOO=a=b"]`, map[string]string{"FOO": "a=b"}},
		{"list bare key reads agent env", `["FROM_AGENT", "MISSING"]`, map[string]string{"FROM_AGENT": "agent-value", "MISSING": ""}},
		{"list skips empty key", `["=value"]`, map[string]string{}},
		{
			"map",
			`{"FOO": "bar", "PRIORITY": -1, "LARGE": 12345678901234567891, "DEBUG": true}`,
			map[string]string{"FOO": "bar", "PRIORITY": "-1", "LARGE": "12345678901234567891", "DEBUG": "true"},
		},
		{"map null reads agent env", `{"FROM_AGENT": null}`, map[string]string{"FROM_AGENT": "agent-value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEnv(json.RawMessage(tt.raw))
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseEnvRejectsInvalidShapes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"string", `"FOO=bar"`},
		{"list with non-string entry", `[1]`},
		{"map with nested map", `{"FOO": {"x": 1}}`},
		{"map with list value", `{"FOO": ["a"]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseEnv(json.RawMessage(tt.raw))
			assert.Error(t, err)
		})
	}
}

func TestPluginAcceptsMapEnvOnWatchedSteps(t *testing.T) {
	t.Setenv("BUILDKITE_MESSAGE", "some message")
	t.Setenv("BUILDKITE_BRANCH", "some-branch")
	t.Setenv("BUILDKITE_COMMIT", "commit-hash")

	param := `[{
		"github.com/glydways/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "foo-service/",
					"config": {
						"command": "echo foo",
						"env": { "OWNER_SLACK_GROUP": "@data-platform-triage" }
					}
				},
				{
					"path": "bar-service/",
					"config": {
						"trigger": "bar-pipeline",
						"build": { "env": { "JOB_FILTER": "atp_unit_tests", "PRIORITY": "-1" } }
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)

	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"OWNER_SLACK_GROUP": "@data-platform-triage"}, got.Watch[0].Step.Env)
	assert.Equal(t, map[string]string{"JOB_FILTER": "atp_unit_tests", "PRIORITY": "-1"}, got.Watch[1].Step.Build.Env)
}

func TestPluginErrorsOnInvalidWatchedStepEnv(t *testing.T) {
	param := `[{
		"github.com/glydways/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "foo-service/",
					"config": {
						"command": "echo foo",
						"env": { "FOO": { "nested": "value" } }
					}
				}
			]
		}
	}]`

	_, err := initializePlugin(param)

	assert.EqualError(t, err, "failed to parse plugin configuration")
}
