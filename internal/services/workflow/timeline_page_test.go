package workflow

import (
	"testing"

	"github.com/yottaapp/yotta/internal/run"
)

func TestTimelinePageReturnsBoundedNewestFirstPages(t *testing.T) {
	entries := make([]run.JournalEntry, 450)
	for index := range entries {
		entries[index].Sequence = uint64(index + 1)
	}
	latest, page, pages := timelinePage(entries, 1, 200)
	if page != 1 || pages != 3 || len(latest) != 200 || latest[0].Sequence != 251 || latest[199].Sequence != 450 {
		t.Fatalf("latest page=%d pages=%d entries=%d range=%d..%d", page, pages, len(latest), latest[0].Sequence, latest[len(latest)-1].Sequence)
	}
	oldest, page, pages := timelinePage(entries, 3, 200)
	if page != 3 || pages != 3 || len(oldest) != 50 || oldest[0].Sequence != 1 || oldest[49].Sequence != 50 {
		t.Fatalf("oldest page=%d pages=%d entries=%d", page, pages, len(oldest))
	}
}
