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
	entryOpenedMsg     struct{ entry *api.Entry }
	entryEditLoadedMsg struct{ entry *api.Entry }
	entryCreatedMsg    struct{ meta *api.EntryMeta }
	entryUpdatedMsg    struct{ meta *api.EntryMeta }
	entryMovedMsg      struct {
		id     string
		folder string
	}
	entryDeletedMsg struct{ id string }
)
