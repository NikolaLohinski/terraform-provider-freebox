package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"
)

const (
	defaultAppID      = "terraform-provider-freebox"
	defaultDeviceName = "terraform"
)

func NewAuthorization(version string) *model {
	// Endpoint
	endpoint := textinput.New()
	endpoint.Prompt = "  > "
	endpoint.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF"))
	endpoint.Focus()
	endpoint.CharLimit = 1024
	endpoint.SetValue(defaultEndpoint)

	// Version
	appVersion := textinput.New()
	appVersion.Prompt = endpoint.Prompt
	appVersion.PromptStyle = endpoint.PromptStyle
	appVersion.CharLimit = 32
	appVersion.SetValue(version)

	// Application ID
	appID := textinput.New()
	appID.Prompt = endpoint.Prompt
	appID.PromptStyle = endpoint.PromptStyle
	appID.CharLimit = 64
	appID.SetValue(defaultAppID)

	// Device Name
	deviceName := textinput.New()
	deviceName.Prompt = endpoint.Prompt
	deviceName.PromptStyle = endpoint.PromptStyle
	deviceName.CharLimit = 64
	deviceName.SetValue(defaultDeviceName)

	// Spinner
	spin := spinner.New()
	spin.Style = endpoint.PromptStyle
	spin.Spinner = spinner.MiniDot

	return &model{
		spinner:    spin,
		active:     0,
		endpoint:   endpoint,
		version:    appVersion,
		appID:      appID,
		deviceName: deviceName,
		once:       &sync.Once{},
	}
}

type model struct {
	active     int
	endpoint   textinput.Model // active = 0
	version    textinput.Model // active = 1
	appID      textinput.Model // active = 2
	deviceName textinput.Model // active = 3
	spinner    spinner.Model   // active = 4
	token      string          // active = 5
	error      error
	once       *sync.Once
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.error != nil {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.active < 4 {
				m.active += 1
			}
			switch m.active {
			case 0:
				m.endpoint.Focus()
			case 1:
				m.appID.Focus()
			case 2:
				m.version.Focus()
			case 3:
				m.deviceName.Focus()
			case 4:
				m.once.Do(func() {
					go func() {
						time.Sleep(time.Second * 2)
						freebox, err := client.New(m.endpoint.Value(), defaultVersion)
						if err != nil {
							m.error = err
							return
						}
						token, err := freebox.WithAppID(m.appID.Value()).Authorize(context.Background(), types.AuthorizationRequest{
							Name:    m.appID.Value(),
							Version: m.version.Value(),
							Device:  m.deviceName.Value(),
						})
						if err != nil {
							m.error = err
							return
						}
						m.active += 1
						m.token = token
					}()
				})
				return m, m.spinner.Tick
			case 5:
				return m, nil
			default:
				return m, tea.Quit
			}
			return m, textinput.Blink
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case error:
		return m, tea.Quit
	}
	switch m.active {
	case 0:
		m.endpoint, cmd = m.endpoint.Update(msg)
	case 1:
		m.appID, cmd = m.appID.Update(msg)
	case 2:
		m.version, cmd = m.version.Update(msg)
	case 3:
		m.deviceName, cmd = m.deviceName.Update(msg)
	case 4:
		m.spinner, cmd = m.spinner.Update(msg)
	case 5:
		m.active += 1
		m.spinner, cmd = m.spinner.Update(msg)
	default:
		return m, tea.Quit
	}

	return m, cmd
}

func (m *model) View() (s string) {
	var view string
	endpoint := fmt.Sprintf(
		"%s Which endpoint should be used to access your freebox:\n\n%s",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Render("?"),
		m.endpoint.View(),
	)
	if m.active > 0 {
		endpoint = fmt.Sprintf(
			"%s %s:    %s",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00CC00")).Render("✔"),
			environmentVariableEndpoint,
			m.endpoint.Value(),
		)
	}
	view = endpoint

	if m.active > 0 {
		appID := fmt.Sprintf(
			"%s What is the name of your application:\n\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Render("?"),
			m.appID.View(),
		)
		if m.active > 1 {
			appID = fmt.Sprintf(
				"%s %s:      %s",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00CC00")).Render("✔"),
				environmentVariableAppID,
				m.appID.Value(),
			)
		}
		view += "\n" + appID
	}

	if m.active > 1 {
		version := fmt.Sprintf(
			"%s What is the version of your application:\n\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Render("?"),
			m.version.View(),
		)
		if m.active > 2 {
			version = fmt.Sprintf(
				"%s FREEBOX_APP_VERSION: %s",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00CC00")).Render("✔"),
				m.version.Value(),
			)
		}
		view += "\n" + version
	}

	if m.active > 2 {
		deviceName := fmt.Sprintf(
			"%s Pick a name for the device that will run your application:\n\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Render("?"),
			m.deviceName.View(),
		)
		if m.active > 3 {
			deviceName = fmt.Sprintf(
				"%s FREEBOX_DEVICE:      %s",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00CC00")).Render("✔"),
				m.deviceName.Value(),
			)
		}
		view += "\n" + deviceName
	}

	if m.active == 4 {
		view += "\n" + fmt.Sprintf(
			"%s Please head to your Freebox Server and click on %s\n   %s Waiting for the request to be approved...",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Render(">"),
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render("✔"),
			m.spinner.View(),
		)
	}
	if m.active > 5 {
		view += "\n" + fmt.Sprintf(
			"%s %s:       %s",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00CC00")).Render("✔"),
			environmentVariableToken,
			m.token,
		)
		return view + "\n"
	}
	if m.error != nil {
		view += "\n" +
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000")).Render("× "+m.error.Error())
		return view + "\n"
	}
	return view + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n\n(ESC or CTLR+C to quit)\n")
}
