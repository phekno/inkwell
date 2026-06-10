package ui

import (
	"github.com/phekno/inkwell/tui/internal/api"
	"github.com/phekno/inkwell/tui/internal/cognito"
)

type (
	signedInMsg  struct{ session *cognito.Session }
	signInErrMsg struct{ err error }
)

type (
	entriesLoadedMsg struct{ entries []api.EntryMeta }
	entriesErrMsg    struct{ err error }
)

type (
	entryOpenedMsg  struct{ entry *api.Entry }
	entryCreatedMsg struct{ meta *api.EntryMeta }
	entryDeletedMsg struct{ id string }
)
