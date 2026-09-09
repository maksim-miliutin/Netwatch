package takeout

import (
	"strings"
	"testing"
)

const sample = `[
  {
    "header": "YouTube",
    "title": "Watched Нечто длинное",
    "titleUrl": "https://www.youtube.com/watch?v=abc123",
    "time": "2026-08-01T12:00:00.000Z"
  },
  {
    "header": "YouTube",
    "title": "Watched a video that has been removed",
    "time": "2026-08-02T12:00:00.000Z"
  },
  {
    "header": "YouTube",
    "title": "Searched for кошки",
    "titleUrl": "https://www.youtube.com/results?search_query=кошки",
    "time": "2026-08-03T12:00:00.000Z"
  }
]`

func TestReadsWhatItCanUse(t *testing.T) {
	plays, skipped, err := Read(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}

	if len(plays) != 1 {
		t.Fatalf("read %d, wanted the one that is a video", len(plays))
	}

	if plays[0].ID != "abc123" {
		t.Errorf("got id %q", plays[0].ID)
	}

	// A video taken down and a search are both rows this cannot use, and a
	// file that yields nothing should say so rather than look empty.
	if skipped != 2 {
		t.Errorf("skipped %d, wanted two", skipped)
	}
}

// Google writes "Watched " in front of the name, in whatever language the
// account was set to.
func TestCutsTheWordGooglePutsInFront(t *testing.T) {
	plays, _, _ := Read(strings.NewReader(sample))

	if plays[0].Title != "Нечто длинное" {
		t.Errorf("got %q, wanted the name without the word in front", plays[0].Title)
	}
}

func TestRefusesAFileThatIsNotOne(t *testing.T) {
	if _, _, err := Read(strings.NewReader("это не json")); err == nil {
		t.Error("read it anyway")
	}
}

func TestReadsAnEmptyHistoryAsEmpty(t *testing.T) {
	plays, skipped, err := Read(strings.NewReader("[]"))

	if err != nil || len(plays) != 0 || skipped != 0 {
		t.Errorf("got %d plays, %d skipped, %v", len(plays), skipped, err)
	}
}
