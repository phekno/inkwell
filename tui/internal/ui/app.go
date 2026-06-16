package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/phekno/inkwell/tui/internal/api"
	"github.com/phekno/inkwell/tui/internal/cognito"
	"github.com/phekno/inkwell/tui/internal/config"
)

type screen int

const (
	screenLogin screen = iota
	screenEntries
)

type App struct {
	cfg     config.Config
	cognito *cognito.Client
	session *cognito.Session

	screen  screen
	login   loginModel
	entries entriesModel

	width, height int
}

func New(ctx context.Context) (App, error) {
	// Detect the terminal theme once, here, before tea.Program starts reading
	// stdin — querying it later (e.g. via glamour) would block. See glamourStyle.
	if !lipgloss.HasDarkBackground() {
		glamourStyle = "light"
	}

	cfg := config.Load()
	cog, err := cognito.New(ctx, cognito.Config{
		Region:     cfg.Region,
		UserPoolID: cfg.UserPoolID,
		ClientID:   cfg.ClientID,
	})
	if err != nil {
		return App{}, err
	}

	a := App{
		cfg:     cfg,
		cognito: cog,
		screen:  screenLogin,
		login:   newLogin(cog),
	}

	// Restore an existing session if one is on disk. Cognito tokens last only
	// about an hour, so refresh first when expired; if the refresh token is
	// also dead, fall through to the login screen instead of carrying a token
	// the API will reject.
	if s, _ := cognito.LoadSession(); s != nil {
		if s.Expired() {
			if err := cog.Refresh(ctx, s); err == nil {
				_ = cognito.SaveSession(s)
			} else {
				s = nil
			}
		}
		if s != nil {
			a.session = s
			a.screen = screenEntries
			a.entries = newEntries(a.newClient(s))
		}
	}
	return a, nil
}

// newClient builds an API client that transparently refreshes the Cognito
// session (and persists it) whenever a request comes back 401.
func (a App) newClient(s *cognito.Session) *api.Client {
	c := api.New(a.cfg.APIURL, s.IDToken)
	cog := a.cognito
	c.Refresh = func() (string, error) {
		if err := cog.Refresh(context.Background(), s); err != nil {
			return "", err
		}
		_ = cognito.SaveSession(s)
		return s.IDToken, nil
	}
	return c
}

func (a App) Init() tea.Cmd {
	if a.screen == screenEntries {
		return a.entries.Init()
	}
	return a.login.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		if a.screen == screenEntries && a.entries.mode == modeList && msg.String() == "q" {
			return a, tea.Quit
		}
	case signedInMsg:
		a.session = msg.session
		a.entries = newEntries(a.newClient(msg.session))
		a.screen = screenEntries
		// Push window size into the new model and kick off the list load.
		sized, _ := a.entries.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		a.entries = sized
		return a, a.entries.Init()
	}

	var cmd tea.Cmd
	switch a.screen {
	case screenLogin:
		a.login, cmd = a.login.Update(msg)
	case screenEntries:
		a.entries, cmd = a.entries.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	body := ""
	switch a.screen {
	case screenLogin:
		body = a.login.View()
	case screenEntries:
		body = a.entries.View()
	}
	status := ""
	if a.session != nil {
		status = "\n\n" + lipgloss.NewStyle().Faint(true).Render("signed in as "+a.session.Email)
	}
	return strings.TrimRight(body, "\n") + status
}
