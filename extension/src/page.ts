// What nothing outside the page can see: whether it is running, how far in, and
// what it is really called.

const EVERY_MS = 10000;

const SITES = 'YouTube|RUTUBE|Rutube|VK Видео|ВКонтакте|Twitch|Дзен|ОК|Кинопоиск'
    + '|Okko|ivi|Wink|Premier|Netflix|Vimeo|Dailymotion|Coub|Яндекс Музыка';

const TAIL = new RegExp(`\\s*[-—|]\\s*(${SITES})\\s*$`);

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

// Yandex Music keeps no video or audio element on the page at all: the sound
// comes from somewhere the page cannot be asked about, and mediaSession is the
// only thing that knows. No element means no position and no length either.
function sounding(): { paused: boolean; position: number; length: number } | null
{
    const one = media();

    if (one)
    {
        return {
            paused: one.paused,
            position: one.currentTime,
            length: Number.isFinite(one.duration) ? one.duration : 0,
        };
    }

    const state = navigator.mediaSession.playbackState;

    if (navigator.mediaSession.metadata && state !== 'none')
    {
        return { paused: state === 'paused', position: 0, length: 0 };
    }

    return null;
}

function say(): void
{
    const on = sounding();

    if (!on)
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

    send({ watching: { url: location.href, title, by, ...on } });
}

// Media events do not travel up the page, so they are caught on the way down.
for (const event of ['play', 'pause', 'seeked', 'ended'])
{
    document.addEventListener(event, say, true);
}

window.addEventListener('pagehide', () => send({ gone: true }));

setInterval(say, EVERY_MS);
say();
