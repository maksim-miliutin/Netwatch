package discord

import (
	"strings"
	"testing"
	"time"

	"netwatch/internal/now"
	"netwatch/internal/play"
)

func live(service, title string) now.Live {
	return now.Live{
		Play: play.Play{
			Service: service,
			ID:      "abc",
			Title:   title,
			URL:     "https://youtu.be/abc",
		},
		Started: time.Now().Add(-time.Minute),
		Ends:    time.Now().Add(9 * time.Minute),
	}
}

func TestPutsTheNameOnTheCard(t *testing.T) {
	card := Card(live("youtube", "Нечто длинное"))

	if card.Details != "Нечто длинное" {
		t.Errorf("first line is %q", card.Details)
	}

	if card.Assets.Large != "youtube" {
		t.Errorf("picture is %q", card.Assets.Large)
	}
}

// The word above the card is the one thing everybody reads, and music that
// says "watching" is noticed immediately.
func TestSaysListeningForWhatIsListenedTo(t *testing.T) {
	if kind := Card(live("yandex-music", "Нечто")).Type; kind != listening {
		t.Errorf("got %d, wanted %d", kind, listening)
	}

	if kind := Card(live("youtube", "Нечто")).Type; kind != watching {
		t.Errorf("got %d, wanted %d", kind, watching)
	}
}

func TestNamesTheChannelWhenThePageSaidOne(t *testing.T) {
	one := live("youtube", "Нечто")
	one.By = "Канал"

	if state := Card(one).State; state != "Канал" {
		t.Errorf("second line is %q", state)
	}
}

func TestNamesTheServiceWhenNobodySaidWhose(t *testing.T) {
	if state := Card(live("rutube", "Нечто")).State; state != "RuTube" {
		t.Errorf("second line is %q", state)
	}
}

// A bar that stands still reads as a bar that is stuck.
func TestAPausedCardHasNoBar(t *testing.T) {
	one := live("youtube", "Нечто")
	one.Paused = true

	card := Card(one)

	if card.Timestamps != nil {
		t.Error("a paused card is still counting")
	}

	if card.State != paused {
		t.Errorf("second line is %q", card.State)
	}
}

// A stream has no end, and Discord counts up rather than down when there is
// none to count towards.
func TestAStreamCountsUp(t *testing.T) {
	one := live("twitch", "Кто-то")
	one.Ends = time.Time{}

	stamps := Card(one).Timestamps
	if stamps == nil || stamps.Start == 0 {
		t.Fatalf("got %+v", stamps)
	}

	if stamps.End != 0 {
		t.Errorf("a stream ends at %d", stamps.End)
	}
}

// Discord cuts a long line itself, mid-word and without saying it did.
func TestCutsALineThatIsTooLong(t *testing.T) {
	card := Card(live("youtube", strings.Repeat("я", 300)))

	if got := len([]rune(card.Details)); got != longest {
		t.Errorf("kept %d characters, wanted %d", got, longest)
	}

	if !strings.HasSuffix(card.Details, "…") {
		t.Errorf("cut to %q without saying so", card.Details)
	}
}

func TestOffersTheAddressAsAButton(t *testing.T) {
	card := Card(live("youtube", "Нечто"))

	if len(card.Buttons) != 1 || card.Buttons[0].URL != "https://youtu.be/abc" {
		t.Errorf("got %+v", card.Buttons)
	}
}

// Discord refuses a button that does not point at the web, and takes the
// whole card down with it rather than just the button.
func TestLeavesOutAButtonThatGoesNowhere(t *testing.T) {
	one := live("youtube", "Нечто")
	one.Play.URL = "javascript:void(0)"

	if buttons := Card(one).Buttons; len(buttons) != 0 {
		t.Errorf("got %+v", buttons)
	}
}
