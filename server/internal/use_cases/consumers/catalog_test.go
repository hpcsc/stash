//go:build unit

package consumers_test

import (
	"testing"

	"github.com/hpcsc/stash/internal/use_cases/consumers"
	"github.com/stretchr/testify/require"
)

func TestDescriptions(t *testing.T) {
	byName := make(map[string]consumers.Description, len(consumers.Descriptions()))
	for _, description := range consumers.Descriptions() {
		byName[description.Name] = description
	}

	require.Equal(t, consumers.KindProjection, byName["items"].Kind)
	require.Equal(t, consumers.KindProjection, byName["tags"].Kind)
	require.Equal(t, consumers.KindProjection, byName["item-tags"].Kind)
	require.Equal(t, consumers.KindReaction, byName["link-metadata"].Kind)
	require.Equal(t, consumers.KindReaction, byName["thumbnail"].Kind)
	require.Equal(t, consumers.KindReaction, byName["empty-trash"].Kind)

	for _, name := range []string{"items", "tags", "item-tags", "link-metadata", "thumbnail", "empty-trash"} {
		require.NotEmpty(t, byName[name].EventTypes, "%s must list the event types it reads", name)
	}
}
