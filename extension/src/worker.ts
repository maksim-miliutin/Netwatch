// Tells netwatch what is open and nothing anywhere else. The port is asked
// for every time: a service worker sleeps all day and wakes up out of date.
async function netwatch(): Promise<string>
{
    const kept = await chrome.storage.local.get('port');

    return 'http://127.0.0.1:' + (kept.port ?? 7373) + '/api';
}

const SETTLE_MS = 4000;

const HOLDS_MS = 30000;

interface Watching
{
    url: string;
    title: string;
    by: string;
    paused: boolean;
    position: number;
    length: number;
}

type Said = { gone: true } | { watching: Watching };

const waiting = new Map<number, number>();

let playing: number | null = null;
let heard = 0;

// Nothing is written down while this is on. Not everything watched is
// something to keep, and the only honest way to leave it out is not to send it.
async function asleep(): Promise<boolean>
{
    const kept = await chrome.storage.local.get('asleep');

    return kept.asleep === true;
}

async function post(where: string, said: unknown): Promise<void>
{
    if (await asleep())
    {
        return;
    }

    try
    {
        await fetch(await netwatch() + where,
        {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify(said),
        });
    }
    catch
    {
        // netwatch is not running, and a browser should not complain about a
        // program somebody closed.
    }
}

function tell(tab: chrome.tabs.Tab): void
{
    if (tab.id === undefined || !tab.url || !tab.url.startsWith('http'))
    {
        return;
    }

    const id = tab.id;
    const url = tab.url;

    clearTimeout(waiting.get(id));

    waiting.set(id, setTimeout(() =>
    {
        waiting.delete(id);
        post('/seen', { url, title: tab.title ?? '' });
    }, SETTLE_MS));
}

chrome.runtime.onMessage.addListener((said: Said, from: chrome.runtime.MessageSender) =>
{
    const tab = from.tab?.id ?? null;
    const at = Date.now();

    if ('gone' in said)
    {
        if (tab !== playing)
        {
            return;
        }

        playing = null;
        post('/gone', {});

        return;
    }

    // Whichever tab started first keeps the card. A second video opened beside
    // it waits rather than taking over halfway through the first.
    if (playing !== null && playing !== tab && at - heard < HOLDS_MS)
    {
        return;
    }

    playing = tab;
    heard = at;

    post('/now', said.watching);
});

chrome.tabs.onUpdated.addListener((id, changed, tab) =>
{
    if (changed.url || changed.title)
    {
        tell(tab);
    }
});

chrome.tabs.onRemoved.addListener((id) =>
{
    clearTimeout(waiting.get(id));
    waiting.delete(id);

    if (id === playing)
    {
        playing = null;
        post('/gone', {});
    }
});
