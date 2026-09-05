# netwatch

A list of what was watched and listened to on this machine. YouTube, RuTube,
VK Video, Dzen, OK, Twitch, Kick, VK Play, Kinopoisk, Okko, ivi, Wink, Premier,
Netflix, Vimeo, Dailymotion, Coub, Yandex Music — and whatever else you add.

Nothing goes anywhere: the program listens on loopback only, and the list sits
in a file beside it.

## Running

```
go build -o netwatch .
./netwatch
```

One file, ten megabytes, nothing to install. Open `http://127.0.0.1:7373`.

## The extension

The browser is the only one who knows the name of a video. It cannot be had off
the wire: everything is inside TLS, and all that shows is a server address and
the size of the packets.

It is written in TypeScript, so it has to be built first:

```
cd extension
npm install
npm run build
```

Chrome → `chrome://extensions` → Developer mode → Load unpacked → the
`extension` folder.

Two parts. `worker.ts` reads the address and title of a tab, which is what the
list is made of. `page.ts` runs on the service's own pages and knows what a tab
cannot: whether the video is running, how far into it the page has got, and
what it is really called. "Something — YouTube" is the name of a tab, not the
name of a video.

Whichever tab started playing first keeps the line at the top of the page. A
second video opened beside it waits rather than taking over halfway through.

Everything goes to `127.0.0.1:7373` and nowhere else. The manifest says so, and
the browser holds it to that.

## Adding a service

One entry in the `services` list in `internal/play/play.go`:

```go
{
    Name:  "goodgame",
    Shown: "GoodGame",
    Hosts: []string{"goodgame.ru"},
    Watching: func(u *url.URL) string {
        return after(u.Path, "/channel/")
    },
},
```

Nothing else in the program: no branch in a `switch`, no interface, no
registration. `Watching` answers what is being watched at that address, and an
empty string when the service's page is not a watch at all. A search on YouTube
is still YouTube and still not a video. `Shown` is the name people read, and
`Heard: true` is for what somebody listens to rather than watches.

One place outside the program: the address goes into `content_scripts` in the
extension manifest. A manifest is read before the program starts and cannot ask
it anything.

## How long

A play is closed by the next one — the only end most of them get. A tab knows
when it opened and almost never when it was abandoned.

Two consequences, both visible on the page. The last play is always running and
uncounted. And a tab left overnight is not counted at all: nobody watched nine
hours of YouTube, and a list that says so is worse than one that says nothing.

## History from before

The extension sees only what was opened since it was installed. The rest the
service hands over itself.

Google: `takeout.google.com` → YouTube only → history → JSON. The archive holds
`watch-history.json`.

```
./netwatch -import watch-history.json
```

Running it twice costs nothing: the same thing is not written again. Whoever is
unsure whether it worked will run it again, and that is right.

Rows without an address are skipped — a video taken down has nothing to point
at. So are searches: a page of the service, not a watch.

## Next

- How long for real: the page already says when it was left
- Days as well as weeks
- Yandex Music hands over a history through a key of its own
- What plays, on a Discord card
