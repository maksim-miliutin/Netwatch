interface Now
{
    playing: boolean;
    service?: string;
    title?: string;
    by?: string;
    paused?: boolean;
    gone?: number;
    whole?: number;
    discord?: string;
}

const state = document.getElementById('state') as HTMLElement;

function line(text: string, kind: string): void
{
    const said = document.createElement('p');

    said.className = kind;
    said.textContent = text;

    state.append(said);
}

function clock(seconds: number): string
{
    return Math.floor(seconds / 60) + ':' + String(Math.floor(seconds % 60)).padStart(2, '0');
}

function bar(gone: number, whole: number): void
{
    const track = document.createElement('div');
    const fill = document.createElement('i');

    track.className = 'bar';
    fill.style.width = (gone / whole) * 100 + '%';
    track.append(fill);

    const times = document.createElement('p');
    const left = document.createElement('span');
    const right = document.createElement('span');

    times.className = 'times';
    left.textContent = clock(gone);
    right.textContent = clock(whole);
    times.append(left, right);

    state.append(track, times);
}

function draw(on: Now): void
{
    if (!on.playing)
    {
        line('Running. Nothing playing.', 'quiet');
        card(on);

        return;
    }

    line(on.title ?? '', 'what');
    line((on.by ? on.by + ' · ' : '') + on.service + (on.paused ? ' · paused' : ''), 'who');

    if (on.whole && !on.paused)
    {
        bar(on.gone ?? 0, on.whole);
    }

    card(on);
}

// Started without a console, the program has nowhere to say this out loud.
function card(on: Now): void
{
    if (on.discord)
    {
        line(on.discord, 'said');
    }
}

async function ask(): Promise<void>
{
    try
    {
        const kept = await chrome.storage.local.get('port');
        const port = kept.port ?? 7373;

        const answer = await fetch('http://127.0.0.1:' + port + '/api/now');

        draw(await answer.json() as Now);
    }
    catch
    {
        line('netwatch is not running.', 'quiet');
    }
}

chrome.storage.local.get('port').then((kept) =>
{
    const list = document.getElementById('list') as HTMLAnchorElement;

    list.href = 'http://127.0.0.1:' + (kept.port ?? 7373);
});

const pause = document.getElementById('pause') as HTMLButtonElement;

function wording(asleep: boolean): void
{
    pause.textContent = asleep ? 'Start recording' : 'Pause recording';
    document.body.classList.toggle('asleep', asleep);
}

chrome.storage.local.get('asleep').then((kept) =>
{
    const asleep = kept.asleep === true;

    wording(asleep);

    if (asleep)
    {
        line('Paused. Nothing is being written down.', 'stopped');
    }
});

pause.addEventListener('click', async () =>
{
    const kept = await chrome.storage.local.get('asleep');
    const asleep = kept.asleep !== true;

    await chrome.storage.local.set({ asleep });
    wording(asleep);

    state.textContent = '';

    if (asleep)
    {
        line('Paused. Nothing is being written down.', 'stopped');
    }
    else
    {
        ask();
    }
});

ask();
