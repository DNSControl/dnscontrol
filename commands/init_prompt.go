package commands

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

// Asker is the minimal set of interactive prompts used by `dnscontrol
// init`. Tests provide a stub implementation to drive the flow
// deterministically.
type Asker interface {
	// Select asks the user to pick one value from a list. The default is
	// suggested and returned when the user accepts without typing.
	Select(message, help string, options []string, defaultOption string) (string, error)
	// MultiSelect asks the user to pick zero or more values from a list.
	MultiSelect(message, help string, options []string) ([]string, error)
	// Input asks for a free form string.
	Input(message, help, defaultValue string) (string, error)
	// Secret asks for a string and masks the input.
	Secret(message, help string) (string, error)
	// Multiline opens an external editor so the user can enter a value
	// that contains newlines (for example a PEM encoded private key).
	Multiline(message, help string) (string, error)
	// Confirm asks a yes/no question.
	Confirm(message string, defaultValue bool) (bool, error)
}

// surveyAsker is the default Asker backed by github.com/AlecAivazis/survey/v2.
type surveyAsker struct{}

// Select implements Asker.
func (surveyAsker) Select(message, help string, options []string, defaultOption string) (string, error) {
	var answer string
	prompt := &survey.Select{
		Message:  message,
		Options:  options,
		Help:     help,
		PageSize: 15,
	}
	// Survey rejects a Default that is not one of the options. Only set
	// it when it matches, otherwise fall back to the first option.
	if slices.Contains(options, defaultOption) {
		prompt.Default = defaultOption
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}
	return answer, nil
}

// MultiSelect implements Asker.
func (surveyAsker) MultiSelect(message, help string, options []string) ([]string, error) {
	var answers []string
	prompt := &survey.MultiSelect{
		Message:  message,
		Options:  options,
		Help:     help,
		PageSize: 15,
	}
	if err := survey.AskOne(prompt, &answers); err != nil {
		return nil, err
	}
	return answers, nil
}

// Input implements Asker.
func (surveyAsker) Input(message, help, defaultValue string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: message,
		Default: defaultValue,
		Help:    help,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}
	return answer, nil
}

// Secret implements Asker.
func (surveyAsker) Secret(message, help string) (string, error) {
	var answer string
	prompt := &survey.Password{Message: message, Help: help}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}
	return answer, nil
}

// Multiline implements Asker.
func (surveyAsker) Multiline(message, help string) (string, error) {
	var answer string
	prompt := &survey.Editor{
		Message:       message,
		Help:          help,
		HideDefault:   true,
		AppendDefault: true,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}
	return answer, nil
}

// Confirm implements Asker.
func (surveyAsker) Confirm(message string, defaultValue bool) (bool, error) {
	var answer bool
	prompt := &survey.Confirm{Message: message, Default: defaultValue}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return false, err
	}
	return answer, nil
}

// askField prompts for a single CredsField and returns the value the user
// entered, respecting Required, Secret, Default and Choices.
func askField(asker Asker, field providers.CredsField) (string, error) {
	defaultValue := field.Default
	if field.EnvVar != "" {
		if envValue := os.Getenv(field.EnvVar); envValue != "" {
			defaultValue = envValue
		}
	}

	label := field.Label
	if label == "" {
		label = field.Key
	} else if !strings.EqualFold(label, field.Key) {
		label += " [" + field.Key + "]"
	}
	if field.Required {
		label += " (required)"
	} else {
		label += " (optional)"
	}

	for {
		var (
			value string
			err   error
		)
		switch {
		case len(field.Choices) > 0:
			value, err = asker.Select(label, field.Help, field.Choices, defaultValue)
		case field.Multiline:
			value, err = asker.Multiline(label, field.Help)
		case field.Secret:
			value, err = asker.Secret(label, field.Help)
		default:
			value, err = asker.Input(label, field.Help, defaultValue)
		}
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value == "" && field.Required {
			fmt.Fprintln(os.Stderr, "A value is required.")
			continue
		}
		if field.Validator != nil && value != "" {
			if err := field.Validator(value); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				continue
			}
		}
		return value, nil
	}
}

// openPortalHint prints the portal URL plus any provider notes so the
// user can open the link themselves before answering the credential
// prompts.
func openPortalHint(_ Asker, meta providers.CredsMetadata) error {
	if meta.PortalURL == "" && meta.Notes == "" {
		return nil
	}
	fmt.Println()
	if meta.PortalURL != "" {
		fmt.Printf("API settings for %s: %s\n", displayName(meta.TypeName), meta.PortalURL)
	}
	if meta.Notes != "" {
		fmt.Println(meta.Notes)
	}
	return nil
}

// collectFields runs askField for every field defined in meta and returns
// the resulting key/value map. Fields whose ShowIf condition does not
// match are skipped. Internal fields are not written to the output.
// Empty optional answers are dropped.
func collectFields(asker Asker, meta providers.CredsMetadata) (map[string]string, error) {
	answers := map[string]string{}
	output := map[string]string{}
	for _, field := range meta.Fields {
		if !showField(field, answers) {
			continue
		}
		value, err := askField(asker, field)
		if err != nil {
			return nil, err
		}
		answers[field.Key] = value
		if field.Internal {
			continue
		}
		if value == "" && !field.Required {
			continue
		}
		if value == field.Default && !field.Required {
			continue
		}
		output[field.Key] = value
	}
	return output, nil
}

// showField evaluates the ShowIf map against the already collected
// answers.
func showField(field providers.CredsField, answers map[string]string) bool {
	for key, want := range field.ShowIf {
		if answers[key] != want {
			return false
		}
	}
	return true
}

// displayName returns the human friendly DisplayName registered for the
// given provider type, falling back to the type name itself when no
// metadata or DisplayName is registered.
func displayName(typeName string) string {
	if meta, ok := providers.GetCredsMetadata(typeName); ok && meta.DisplayName != "" {
		return meta.DisplayName
	}
	return typeName
}

// errInitAborted is returned when the user aborts the init flow.
var errInitAborted = errors.New("init aborted by user")
