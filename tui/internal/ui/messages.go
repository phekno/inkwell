package ui

import (
	"github.com/phekno/inkwell/tui/internal/api"
	"github.com/phekno/inkwell/tui/internal/cognito"
)

type signedInMsg struct{ session *cognito.Session }
type signInErrMsg struct{ err error }

type entriesLoadedMsg struct{ entries []api.EntryMeta }
type entriesErrMsg struct{ err error }

type entryOpenedMsg struct{ entry *api.Entry }
type entryCreatedMsg struct{ meta *api.EntryMeta }
type entryDeletedMsg struct{ id string }
