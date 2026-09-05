// netwatch keeps a list of what was watched and listened to on this machine.
//
// Nothing here reads the network: what is being watched lives inside TLS and
// no amount of packet reading will say the name of a video. What does know
// the name is the browser looking at the page, so a small extension reports
// the tab and this writes it down.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"netwatch/internal/discord"
	"netwatch/internal/now"
	"netwatch/internal/store"
	"netwatch/internal/takeout"
	"netwatch/internal/web"
)

func main() {
	port := flag.Int("port", 7373, "where to listen, on this machine only")
	file := flag.String("file", "", "where to keep the list")
	from := flag.String("import", "", "a watch-history.json out of a Takeout archive")
	card := flag.String("discord", "",
		"an application id, to put what plays on a Discord card. off forgets it")
	flag.Parse()

	where, err := chosen(*file)
	if err != nil {
		log.Fatal(err)
	}

	kept, err := store.Open(where)
	if err != nil {
		log.Fatal(err)
	}

	if *from != "" {
		if err := bring(kept, *from); err != nil {
			log.Fatal(err)
		}

		return
	}

	id, err := remembered(*card, filepath.Dir(where))
	if err != nil {
		log.Fatal(err)
	}

	watching := now.New()
	server := &web.Server{Store: kept, Watching: watching}
	address := fmt.Sprintf("127.0.0.1:%d", *port)

	fmt.Printf("netwatch is running. Open http://%s in a browser.\n", address)
	fmt.Printf("The list is kept in %s and goes nowhere else.\n", where)

	// The card is the one thing here that leaves the machine, so it runs only
	// for somebody who went and got an application id for it.
	if id != "" {
		go discord.Follow(context.Background(), id, watching, func(text string) {
			fmt.Println(text)
		})
	}

	log.Fatal(http.ListenAndServe(address, web.Near(server.Routes())))
}

// Reads a history handed over by a service and stops. Importing and serving in
// one run would leave somebody watching a page while the list rearranges under
// them.
func bring(kept *store.Store, from string) error {
	file, err := os.Open(from)
	if err != nil {
		return err
	}
	defer file.Close()

	plays, skipped, err := takeout.Read(file)
	if err != nil {
		return err
	}

	added, err := kept.Merge(plays)
	if err != nil {
		return err
	}

	fmt.Printf("Read %d, kept %d new.\n", len(plays), added)

	if skipped > 0 {
		fmt.Printf("Passed over %d: taken down, or a search rather than a watch.\n",
			skipped)
	}

	if added == 0 && len(plays) > 0 {
		fmt.Println("All of it was already here, which is what a second run should do.")
	}

	return nil
}

// A flag is no use to somebody who starts this by double clicking it, and an
// application id is not a secret: everybody who reads the card can read it.
func remembered(asked, dir string) (string, error) {
	file := filepath.Join(dir, "discord")

	// An id is a number, so the word cannot be one.
	if asked == "off" {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return "", err
		}

		return "", nil
	}

	if asked != "" {
		return asked, os.WriteFile(file, []byte(asked+"\n"), 0o600)
	}

	kept, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return "", nil
	}

	return strings.TrimSpace(string(kept)), err
}

// Beside the program when that can be written to, and in the home folder when
// it cannot: a program dropped in a downloads folder should not lose its list
// the day somebody tidies up.
func chosen(asked string) (string, error) {
	if asked != "" {
		return asked, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".netwatch", "plays.jsonl"), nil
}
