package silenced

import (
	"context"
	"time"

	corev2 "github.com/sensu/core/v2"
	cachev2 "github.com/sensu/sensu-go/backend/store/cache/v2"
	stringsutil "github.com/sensu/sensu-go/util/strings"
)

// AddToSilencedBy takes a silenced entry ID and adds it to a silence of IDs if
// it's not already present in order to avoid duplicated elements
func AddToSilencedBy(id string, ids []string) []string {
	if !stringsutil.InArray(id, ids) {
		ids = append(ids, id)
	}
	return ids
}

type SilencesCache interface {
	Get(namespace string) []cachev2.Value[*corev2.Silenced, corev2.Silenced]
}

// GetSilenced retrieves all silenced entries for a given event, using the
// entity subscription, the check subscription and the check name while
// supporting wildcard silenced entries (e.g. subscription:*)
func GetSilenced(ctx context.Context, event *corev2.Event, cache SilencesCache) {
	if !event.HasCheck() {
		return
	}

	cacheEntries := cache.Get(event.Check.Namespace)
	entries := make([]*corev2.Silenced, 0, len(cacheEntries))
	for _, entry := range cacheEntries {
		silenced := entry.Resource
		if silenced.ExpireAt > 0 && time.Unix(silenced.ExpireAt, 0).Before(time.Now()) {
			// the entry has expired, and is just a stale cache member
			continue
		}
		entries = append(entries, silenced)
	}

	// Determine which entries silence this event
	silencedIDs := SilencedBy(event, entries)

	// Add to the event all silenced entries ID that actually silence it
	event.Check.Silenced = silencedIDs
}

// SilencedBy determines which of the given silenced entries silenced a given
// event and return a list of silenced entry IDs
func SilencedBy(event *corev2.Event, silencedEntries []*corev2.Silenced) []string {
	silencedBy := event.SilencedBy(silencedEntries)
	names := make([]string, 0, len(silencedBy))
	for _, entry := range silencedBy {
		names = AddToSilencedBy(entry.Name, names)
	}
	return names
}
