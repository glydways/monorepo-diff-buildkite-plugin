package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const pluginName = "github.com/glydways/monorepo-diff"

// Plugin buildkite monorepo diff plugin structure
type Plugin struct {
	Diff          string
	Wait          bool
	LogLevel      string `json:"log_level"`
	Interpolation bool
	Hooks         []HookConfig
	Watch         []WatchConfig
	RawEnv        json.RawMessage `json:"env"`
	Env           map[string]string
	RawNotify     []map[string]interface{} `json:"notify" yaml:",omitempty"`
	Notify        []PluginNotify           `yaml:"notify,omitempty"`
	Plugins       interface{}              `yaml:"plugins,omitempty"`
}

// HookConfig Plugin hook configuration
type HookConfig struct {
	Command string
}

// WatchConfig Plugin watch configuration
type WatchConfig struct {
	RawPath interface{} `json:"path"`
	Paths   []string
	Step    Step `json:"config"`
}

type Group struct {
	Label string `yaml:"group"`
	Steps []Step `yaml:"steps"`
}

// GithubStatusNotification is notification config for github_commit_status
type GithubStatusNotification struct {
	Context string `yaml:"context,omitempty"`
}

// PluginNotify is notify configuration for pipeline
type PluginNotify struct {
	Slack        string                   `yaml:"slack,omitempty"`
	Email        string                   `yaml:"email,omitempty"`
	PagerDuty    string                   `yaml:"pagerduty_change_event,omitempty"`
	Webhook      string                   `yaml:"webhook,omitempty"`
	Basecamp     string                   `yaml:"basecamp_campfire,omitempty"`
	GithubStatus GithubStatusNotification `yaml:"github_commit_status,omitempty"`
	Condition    string                   `yaml:"if,omitempty"`
}

// Notify is Buildkite notification definition
type StepNotify struct {
	Slack        string                   `yaml:"slack,omitempty"`
	Basecamp     string                   `yaml:"basecamp_campfire,omitempty"`
	GithubStatus GithubStatusNotification `yaml:"github_commit_status,omitempty"`
	Condition    string                   `yaml:"if,omitempty"`
}

// Step is buildkite pipeline definition
type Step struct {
	Group             string                   `yaml:"group,omitempty"`
	Trigger           string                   `yaml:"trigger,omitempty"`
	Label             string                   `yaml:"label,omitempty"`
	Key               string                   `yaml:"key,omitempty"`
	Identifier        string                   `yaml:"identifier,omitempty"`
	Build             Build                    `yaml:"build,omitempty"`
	Command           interface{}              `yaml:"command,omitempty"`
	Commands          interface{}              `yaml:"commands,omitempty"`
	Agents            Agent                    `yaml:"agents,omitempty"`
	Artifacts         []string                 `yaml:"artifacts,omitempty"`
	RawEnv            json.RawMessage          `json:"env" yaml:",omitempty"`
	Env               map[string]string        `yaml:"env,omitempty"`
	Async             bool                     `yaml:"async,omitempty"`
	SoftFail          interface{}              `json:"soft_fail" yaml:"soft_fail,omitempty"`
	RawNotify         []map[string]interface{} `json:"notify" yaml:",omitempty"`
	Notify            []StepNotify             `yaml:"notify,omitempty"`
	Plugins           interface{}              `yaml:"plugins,omitempty"`
	TimeoutInMinutes  interface{}              `json:"timeout_in_minutes" yaml:"timeout_in_minutes,omitempty"`
	Parallelism       interface{}              `json:"parallelism" yaml:"parallelism,omitempty"`
	Retry             interface{}              `json:"retry" yaml:"retry,omitempty"`
	DependsOn         interface{}              `json:"depends_on" yaml:"depends_on,omitempty"`
	Condition         string                   `json:"if" yaml:"if,omitempty"`
	Branches          interface{}              `json:"branches" yaml:"branches,omitempty"`
	Concurrency       int                      `json:"concurrency" yaml:"concurrency,omitempty"`
	ConcurrencyGroup  string                   `json:"concurrency_group" yaml:"concurrency_group,omitempty"`
	ConcurrencyMethod string                   `json:"concurrency_method" yaml:"concurrency_method,omitempty"`
}

// Agent is Buildkite agent definition
type Agent map[string]string

// Build is buildkite build definition
type Build struct {
	Message  string            `yaml:"message,omitempty"`
	Branch   string            `yaml:"branch,omitempty"`
	Commit   string            `yaml:"commit,omitempty"`
	RawEnv   json.RawMessage   `json:"env" yaml:",omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	MetaData map[string]string `json:"meta_data,omitempty" yaml:"meta_data,omitempty"`
}

// UnmarshalJSON set defaults properties
func (plugin *Plugin) UnmarshalJSON(data []byte) error {
	type plain Plugin

	def := &plain{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
	}

	// TODO: consider returning this error instead of discarding it. A wrongly
	// typed value is currently dropped silently — `concurrency: "1"` decodes to
	// 0 and the step ends up with no limit at all. Going strict surfaces those
	// typos at parse time, but it would start failing builds for downstream
	// users whose configs carry wrongly typed values that are ignored today, so
	// it needs a heads-up and a major version bump.
	_ = json.Unmarshal(data, def)

	*plugin = Plugin(*def)

	parseResult, err := parseEnv(plugin.RawEnv)
	if err != nil {
		return errors.New("failed to parse plugin configuration")
	}

	plugin.Env = parseResult
	plugin.RawEnv = nil

	setPluginNotify(&plugin.Notify, &plugin.RawNotify)

	// Path can be string or an array of strings,
	// handle both cases and create an array of paths.
	for i, p := range plugin.Watch {
		switch p.RawPath.(type) {
		case string:
			plugin.Watch[i].Paths = []string{plugin.Watch[i].RawPath.(string)}
		case []interface{}:
			for _, v := range plugin.Watch[i].RawPath.([]interface{}) {
				plugin.Watch[i].Paths = append(plugin.Watch[i].Paths, v.(string))
			}
		}

		if plugin.Watch[i].Step.Trigger != "" {
			setBuild(&plugin.Watch[i].Step.Build)
		}

		if plugin.Watch[i].Step.RawNotify != nil {
			setNotify(&plugin.Watch[i].Step.Notify, &plugin.Watch[i].Step.RawNotify)
		}

		if err := appendEnv(&plugin.Watch[i], plugin.Env); err != nil {
			return fmt.Errorf("integrating environment specification into plugin watch configuration at index %d: %w", i, err)
		}

		p.RawPath = nil
	}

	return nil
}

func initializePlugin(data string) (Plugin, error) {
	log.Debugf("parsing plugin config: %v", data)

	var pluginConfigs []map[string]json.RawMessage

	if err := json.Unmarshal([]byte(data), &pluginConfigs); err != nil {
		log.Debug(err)
		return Plugin{}, errors.New("failed to parse plugin configuration")
	}

	for _, p := range pluginConfigs {
		for key, pluginConfig := range p {
			if strings.HasPrefix(key, pluginName) {
				var plugin Plugin

				if err := json.Unmarshal(pluginConfig, &plugin); err != nil {
					log.Debug(err)
					return Plugin{}, errors.New("failed to parse plugin configuration")
				}

				return plugin, nil
			}
		}
	}

	return Plugin{}, errors.New("could not initialize plugin")
}

func setPluginNotify(notifications *[]PluginNotify, rawNotify *[]map[string]interface{}) {
	for _, v := range *rawNotify {
		var notify PluginNotify

		if condition, ok := isString(v["if"]); ok {
			notify.Condition = condition
		}

		if email, ok := isString(v["email"]); ok {
			notify.Email = email
			*notifications = append(*notifications, notify)
			continue
		}

		if basecamp, ok := isString(v["basecamp_campfire"]); ok {
			notify.Basecamp = basecamp
			*notifications = append(*notifications, notify)
			continue
		}

		if webhook, ok := isString(v["webhook"]); ok {
			notify.Webhook = webhook
			*notifications = append(*notifications, notify)
			continue
		}

		if pagerduty, ok := isString(v["pagerduty_change_event"]); ok {
			notify.PagerDuty = pagerduty
			*notifications = append(*notifications, notify)
			continue
		}

		if slack, ok := isString(v["slack"]); ok {
			notify.Slack = slack
			*notifications = append(*notifications, notify)
			continue
		}

		if github, ok := v["github_commit_status"].(map[string]interface{}); ok {
			if context, ok := isString(github["context"]); ok {
				notify.GithubStatus = GithubStatusNotification{Context: context}
				*notifications = append(*notifications, notify)
			}
			continue
		}
	}

	*rawNotify = nil
}

func setNotify(notifications *[]StepNotify, rawNotify *[]map[string]interface{}) {
	for _, v := range *rawNotify {
		var notify StepNotify

		if condition, ok := isString(v["if"]); ok {
			notify.Condition = condition
		}

		if basecamp, ok := isString(v["basecamp_campfire"]); ok {
			notify.Basecamp = basecamp
			*notifications = append(*notifications, notify)
			continue
		}

		if slack, ok := isString(v["slack"]); ok {
			notify.Slack = slack
			*notifications = append(*notifications, notify)
			continue
		}

		if github, ok := v["github_commit_status"].(map[string]interface{}); ok {
			if context, ok := isString(github["context"]); ok {
				notify.GithubStatus = GithubStatusNotification{Context: context}
				*notifications = append(*notifications, notify)
			}
			continue
		}
	}

	*rawNotify = nil
}

func setBuild(build *Build) {
	if build.Message == "" {
		build.Message = env("BUILDKITE_MESSAGE", "")
	}

	if build.Branch == "" {
		build.Branch = env("BUILDKITE_BRANCH", "")
	}

	if build.Commit == "" {
		build.Commit = env("BUILDKITE_COMMIT", "")
	}
}

// appendEnv appends a top-level env to Step.Env and Step.Build.Env.
func appendEnv(watch *WatchConfig, env map[string]string) error {
	var err error
	if watch.Step.Env, err = parseEnv(watch.Step.RawEnv); err != nil {
		return fmt.Errorf("parsing raw environment specification from step: %w", err)
	}
	if watch.Step.Build.Env, err = parseEnv(watch.Step.Build.RawEnv); err != nil {
		return fmt.Errorf("parsing raw environment specification from build: %w", err)
	}

	for key, value := range env {
		if watch.Step.Command != nil || watch.Step.Commands != nil {
			if watch.Step.Env == nil {
				watch.Step.Env = make(map[string]string)
			}

			watch.Step.Env[key] = value
			continue
		}
		if watch.Step.Trigger != "" {
			if watch.Step.Build.Env == nil {
				watch.Step.Build.Env = make(map[string]string)
			}

			watch.Step.Build.Env[key] = value
			continue
		}
	}

	watch.Step.RawEnv = nil
	watch.Step.Build.RawEnv = nil
	watch.RawPath = nil
	return nil
}

// parseEnv interprets either a list of "KEY=value" strings or a mapping entry pair as a set of environment variable bindings.
// A bare KEY (or a null map value) takes its value from the agent environment.
func parseEnv(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// UseNumber keeps numeric values as written instead of rounding them through float64.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded interface{}
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("parsing plugin configuration: %w", err)
	}

	switch entries := decoded.(type) {
	case nil:
		return nil, nil
	case map[string]interface{}:
		result := make(map[string]string, len(entries))
		for key, value := range entries {
			parsed, err := parseEnvValue(key, value)
			if err != nil {
				return nil, err
			}
			result[key] = parsed
		}
		return result, nil
	case []interface{}:
		result := make(map[string]string, len(entries))
		for _, entry := range entries {
			pair, ok := entry.(string)
			if !ok {
				return nil, fmt.Errorf("failed to parse plugin configuration: env entry %v must be a string, got %T", entry, entry)
			}

			key, value, hasValue := strings.Cut(pair, "=")
			// Note: Environment variable names cannot contain whitespace
			key = strings.TrimSpace(key)
			if len(key) == 0 {
				continue
			}

			if !hasValue {
				result[key] = env(key, "")
				continue
			}
			result[key] = strings.TrimSpace(value)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("failed to parse plugin configuration: env must be a list or a map, got %T", decoded)
	}
}

func parseEnvValue(key string, value interface{}) (string, error) {
	// The config comes from encoding/json, which decodes every scalar into one of these types.
	switch v := value.(type) {
	case nil:
		return env(key, ""), nil
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case json.Number:
		return v.String(), nil
	default:
		return "", fmt.Errorf("failed to parse plugin configuration: environment value %q must be a scalar, got %T", key, value)
	}
}
