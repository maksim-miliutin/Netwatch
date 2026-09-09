// What nothing outside the page can see: whether it is running, how far in, and
// what it is really called.

const EVERY_MS = 10000;

// What a site puts after the name in a tab title, used when the page did not
// name what it plays itself.
const SITES = 'YouTube|RUTUBE|Rutube|VK Видео|ВКонтакте|Twitch|Дзен|ОК|Кинопоиск'
    + '|Okko|ivi|Wink|Premier|Netflix|Vimeo|Dailymotion|Coub|Яндекс Музыка';

const TAIL = new RegExp(`\\s*[-—|]\\s*(${SITES})\\s*$`);

// What a browser puts in front of a tab title when messages are waiting.
const COUNTED = /^\(\d+\)\s*/;

function named(): { title: string; by: string }
{
    const said = navigator.mediaSession.metadata;

    if (said?.title)
    {
        return { title: said.title, by: said.artist ?? '' };
    }

    return { title: document.title.replace(COUNTED, '').replace(TAIL, ''), by: '' };
}

// A preview may play muted beside the thing somebody came for.
function media(): HTMLMediaElement | null
{
    const all = [...document.querySelectorAll<HTMLMediaElement>('video, audio')];

    return all.find((one) => !one.paused)
        ?? all.find((one) => one.duration > 0)
        ?? null;
}

function send(what: unknown): void
{
    try
    {
        chrome.runtime.sendMessage(what);
    }
    catch
    {
        // The extension was reloaded and this page belongs to the one before it.
    }
}

function say(): void
{
    const one = media();

    if (!one)
    {
        send({ gone: true });

        return;
    }

    const { title, by } = named();

    // Before a page has said what it plays, it is called after the site. The
    // tick comes round in ten seconds, by which time it has, and a card that
    // says "YouTube" is worse than one that waits.
    if (!title || SITES.split('|').includes(title))
    {
        return;
    }

    send(
    {
        watching:
        {
            url: location.href,
            title,
            by,
            paused: one.paused,
            position: one.currentTime,

            // A stream says its length with an Infinity.
            length: Number.isFinite(one.duration) ? one.duration : 0,
        },
    });
}

// Media events do not travel up the page, so they are caught on the way down.
for (const event of ['play', 'pause', 'seeked', 'ended'])
{
    document.addEventListener(event, say, true);
}

// A page left is over at once; the tick would find it half a minute later.
window.addEventListener('pagehide', () => send({ gone: true }));

setInterval(say, EVERY_MS);
say();
