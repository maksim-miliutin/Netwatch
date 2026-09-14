# Privacy

netwatch keeps a list of what was watched and listened to on one machine, and
shows what is playing on a Discord profile. This says what it reads, where it
puts it, and the one thing that leaves the computer.

Last changed 10 September 2026.

## What the extension reads

The address and title of a tab that is playing something, and what the page
itself says it is playing — the name of the track or video and whose it is,
through the browser's own `mediaSession`.

It reads this only on the services listed in `manifest.json`, and on sites
added by hand after the browser has been asked for permission. Nowhere else.

Nothing is read from a tab that is not playing anything.

## Where it goes

To `127.0.0.1` and nowhere else. The manifest permits loopback alone, and the
browser holds the extension to that. The program listening there refuses any
connection that did not come from this machine, and any request that did not
come from its own page.

There is no netwatch server. There is no account, no sign in, no telemetry, no
crash reporting, no analytics of any kind. Nothing is sold, shared or sent
anywhere for any purpose.

## Where it is kept

In a folder beside the program, or in the home folder when that cannot be
written to:

- `plays.jsonl` — the list, one play to a line
- `discord` — the application id the card runs on
- `quiet` — services kept off the card
- `services` — sites added by hand
- `language` — the language the page is in
- `log` — what the program said, kept because a program started without a
  console has nowhere else to say it

Deleting that folder deletes everything netwatch knows. Single plays can be
crossed out from the page, and the whole list taken away as a spreadsheet.

## The one thing that leaves

The Discord card, and only while it is switched on.

netwatch hands the card to the Discord client running on the same machine,
through a socket that belongs to it. Discord then shows it to whoever can see
your profile — that part is Discord's, under
[their privacy policy](https://discord.com/privacy), not ours.

The card carries the name of what is playing, whose it is, the name of the
service, how far along it is, and a button leading to the address. Nothing
else: a card has no history, only this second.

It is off until an application id is put in, services can be kept off it one by
one, and recording can be paused from the extension so that nothing is written
down at all.

## Children

netwatch is not directed at children and collects nothing from anybody,
including them.

## Changes

Any change to this is a commit in
[the repository](https://github.com/maksim-miliutin/Netwatch), with the date
above changed. There is nowhere else it could quietly become untrue.
