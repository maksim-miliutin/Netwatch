package discord

import (
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
)

type said struct {
	opcode uint32
	body   []byte
}

type answer struct {
	opcode uint32
	body   string
}

// A Discord that is not Discord. The reading and writing are the package's own,
// which is the point: a frame this cannot read Discord could not read either.
func fake(answers ...answer) (io.ReadWriteCloser, <-chan said) {
	ours, theirs := net.Pipe()
	heard := make(chan said, len(answers))

	go func() {
		defer theirs.Close()
		defer close(heard)

		far := &Presence{pipe: theirs}

		for _, reply := range answers {
			opcode, body, err := far.read()
			if err != nil {
				return
			}

			heard <- said{opcode, body}

			if err := far.write(reply.opcode, json.RawMessage(reply.body)); err != nil {
				return
			}
		}
	}()

	return ours, heard
}

const ready = `{"cmd":"DISPATCH","evt":"READY","data":{}}`

func TestSaysWhoItIsBeforeAnythingElse(t *testing.T) {
	pipe, heard := fake(answer{frame, ready})

	presence, err := greet(pipe, "424242")
	if err != nil {
		t.Fatal(err)
	}
	defer presence.Close()

	first := <-heard
	if first.opcode != hello {
		t.Errorf("opened with frame %d, wanted the handshake", first.opcode)
	}

	if !strings.Contains(string(first.body), `"client_id":"424242"`) {
		t.Errorf("said %s", first.body)
	}
}

func TestGivesTheReasonAWrongIdWasRefused(t *testing.T) {
	pipe, _ := fake(answer{goodbye, `{"code":4000,"message":"Invalid Client ID"}`})

	_, err := greet(pipe, "not an id")
	if err == nil {
		t.Fatal("a refused id opened anyway")
	}

	if !strings.Contains(err.Error(), "Invalid Client ID") {
		t.Errorf("said %q, which does not say what is wrong", err)
	}
}

func TestSendsTheCard(t *testing.T) {
	pipe, heard := fake(answer{frame, ready}, answer{frame, `{"cmd":"SET_ACTIVITY","data":{}}`})

	presence, err := greet(pipe, "424242")
	if err != nil {
		t.Fatal(err)
	}
	defer presence.Close()

	<-heard

	if err := presence.Show(&Activity{Type: watching, Details: "Нечто"}); err != nil {
		t.Fatal(err)
	}

	sent := string((<-heard).body)

	for _, want := range []string{`"cmd":"SET_ACTIVITY"`, `"details":"Нечто"`, `"pid":`, `"nonce":`} {
		if !strings.Contains(sent, want) {
			t.Errorf("sent %s, without %s", sent, want)
		}
	}
}

func TestTakesTheCardDownWhenGivenNothing(t *testing.T) {
	pipe, heard := fake(answer{frame, ready}, answer{frame, `{"cmd":"SET_ACTIVITY","data":{}}`})

	presence, err := greet(pipe, "424242")
	if err != nil {
		t.Fatal(err)
	}
	defer presence.Close()

	<-heard

	if err := presence.Show(nil); err != nil {
		t.Fatal(err)
	}

	if sent := string((<-heard).body); !strings.Contains(sent, `"activity":null`) {
		t.Errorf("sent %s, which does not clear anything", sent)
	}
}

func TestReadsAComplaintOutOfAnOrdinaryAnswer(t *testing.T) {
	const refused = `{"cmd":"SET_ACTIVITY","evt":"ERROR","data":{"message":"Invalid activity"}}`

	pipe, heard := fake(answer{frame, ready}, answer{frame, refused})

	presence, err := greet(pipe, "424242")
	if err != nil {
		t.Fatal(err)
	}
	defer presence.Close()

	<-heard

	err = presence.Show(&Activity{Type: watching, Details: "Нечто"})
	if err == nil || !strings.Contains(err.Error(), "Invalid activity") {
		t.Errorf("got %v, wanted the complaint", err)
	}

	<-heard
}
