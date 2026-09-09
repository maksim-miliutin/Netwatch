// Package discord tells the Discord client on this machine what is playing.
// The web API cannot set an activity at all; only a program beside it can.
package discord

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	hello   = 0
	frame   = 1
	goodbye = 2
)

// A length past this is a socket belonging to something other than Discord.
const biggest = 64 << 10

// Discord answers every command. One that does not would be waited on for
// good: the loop stops, the card freezes, and nothing ever reconnects.
const Patience = 15 * time.Second

// The word above the card. Music that says "watching" is noticed at once.
const (
	listening = 2
	watching  = 3
)

var ErrNoClient = errors.New("no discord is running on this machine")

// An Activity is the card. The field names are Discord's own.
type Activity struct {
	Type       int         `json:"type"`
	Details    string      `json:"details,omitempty"`
	State      string      `json:"state,omitempty"`
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	Assets     *Assets     `json:"assets,omitempty"`
	Buttons    []Button    `json:"buttons,omitempty"`
}

// Milliseconds. Discord draws the bar off these and keeps it right between updates.
type Timestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

// Large is the name of a picture uploaded to the application, not an address.
type Assets struct {
	Large     string `json:"large_image,omitempty"`
	LargeText string `json:"large_text,omitempty"`
}

type Button struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Presence struct {
	id       string
	pipe     io.ReadWriteCloser
	patience time.Duration

	// Read through a buffer big enough for any frame Discord sends. A named
	// pipe on Windows hands over one message at a time, and asking it for the
	// eight bytes of a head splits a message it will not put back together.
	heard *bufio.Reader

	// One command at a time: every one is answered, and two in flight would
	// each read the other's answer.
	mu sync.Mutex
}

func Open(id string) (*Presence, error) {
	pipe, err := dial()
	if err != nil {
		return nil, err
	}

	return greet(pipe, id)
}

// Apart from Open because a test can be handed both ends of a pipe.
func greet(pipe io.ReadWriteCloser, id string) (*Presence, error) {
	p := &Presence{
		id:       id,
		pipe:     pipe,
		patience: Patience,
		heard:    bufio.NewReaderSize(pipe, biggest),
	}

	if err := p.write(hello, map[string]any{"v": 1, "client_id": id}); err != nil {
		pipe.Close()

		return nil, err
	}

	// The only place a wrong id shows itself, and then only once.
	opcode, body, err := p.hear()
	if err != nil {
		pipe.Close()

		return nil, err
	}

	if opcode == goodbye {
		pipe.Close()

		return nil, refusal(body)
	}

	return p, nil
}

// Nothing takes the card down.
func (p *Presence) Show(card *Activity) error {
	return p.ask("SET_ACTIVITY", map[string]any{"pid": os.Getpid(), "activity": card})
}

func (p *Presence) Close() error {
	return p.pipe.Close()
}

func (p *Presence) ask(command string, args any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	said := map[string]any{"cmd": command, "args": args, "nonce": nonce()}

	if err := p.write(frame, said); err != nil {
		return err
	}

	// Left on the socket the answer fills the pipe, and the next write blocks.
	opcode, body, err := p.hear()
	if err != nil {
		return err
	}

	if opcode == goodbye {
		return refusal(body)
	}

	return complaint(body)
}

// hear is read with a clock on it. The read itself cannot be interrupted, so
// the pipe is closed instead, and that is what brings it back.
func (p *Presence) hear() (uint32, []byte, error) {
	type said struct {
		opcode uint32
		body   []byte
		err    error
	}

	// Room for one, so that a read coming back late has somewhere to put its
	// answer and can finish instead of holding a goroutine forever.
	heard := make(chan said, 1)

	go func() {
		opcode, body, err := p.read()
		heard <- said{opcode, body, err}
	}()

	select {
	case got := <-heard:
		return got.opcode, got.body, got.err

	case <-time.After(p.patience):
		p.pipe.Close()

		return 0, nil, errors.New("discord took the frame and said nothing")
	}
}

func (p *Presence) write(opcode uint32, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	// One write: a Windows pipe makes a message of each, and a head alone is one.
	out := make([]byte, 8, 8+len(payload))
	binary.LittleEndian.PutUint32(out[0:4], opcode)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(payload)))

	_, err = p.pipe.Write(append(out, payload...))

	return err
}

func (p *Presence) read() (uint32, []byte, error) {
	var head [8]byte

	if _, err := io.ReadFull(p.heard, head[:]); err != nil {
		return 0, nil, err
	}

	opcode := binary.LittleEndian.Uint32(head[0:4])
	length := binary.LittleEndian.Uint32(head[4:8])

	if length > biggest {
		return 0, nil, fmt.Errorf("discord sent a frame of %d bytes", length)
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(p.heard, body); err != nil {
		return 0, nil, err
	}

	return opcode, body, nil
}

func refusal(body []byte) error {
	var said struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &said); err != nil || said.Message == "" {
		return errors.New("discord closed the connection without saying why")
	}

	return fmt.Errorf("discord: %s", said.Message)
}

// A card Discord will not show is refused in an ordinary answer, not a close.
func complaint(body []byte) error {
	var said struct {
		Event string `json:"evt"`
		Data  struct {
			Message string `json:"message"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &said); err != nil || said.Event != "ERROR" {
		return nil
	}

	if said.Data.Message == "" {
		return errors.New("discord would not take the card")
	}

	return fmt.Errorf("discord: %s", said.Data.Message)
}

func nonce() string {
	var bytes [8]byte

	if _, err := rand.Read(bytes[:]); err != nil {
		return "0"
	}

	return hex.EncodeToString(bytes[:])
}
