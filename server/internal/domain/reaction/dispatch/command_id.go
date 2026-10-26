package dispatch

import (
	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
)

func CommandIDFor(reactionName string, event *domain.Event) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(reactionName+"/"+event.ID))
}

func CommandIDForTarget(reactionName string, event *domain.Event, target string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(reactionName+"/"+event.ID+"/"+target))
}
