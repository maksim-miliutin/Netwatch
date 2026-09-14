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
        line(said('nothing'), 'quiet');
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

// The language is learnt before anything is said, so nothing is said twice.
async function opening(): Promise<void>
{
    const kept = await api.storage.local.get('port');
    const port = Number(kept.port ?? 7373);

    await learn(port);

    const list = document.getElementById('list') as HTMLAnchorElement;
    list.href = 'http://127.0.0.1:' + port;
    list.textContent = said('open');

    const asleep = (await api.storage.local.get('asleep')).asleep === true;

    wording(asleep);

    if (asleep)
    {
        line(said('paused'), 'stopped');

        return;
    }

    await ask();
}

async function ask(): Promise<void>
{
    try
    {
        const kept = await api.storage.local.get('port');
        const port = kept.port ?? 7373;

        const answer = await fetch('http://127.0.0.1:' + port + '/api/now');

        draw(await answer.json() as Now);
    }
    catch
    {
        line(said('not running'), 'quiet');
    }
}

const pause = document.getElementById('pause') as HTMLButtonElement;

function wording(asleep: boolean): void
{
    pause.textContent = asleep ? said('start') : said('pause');
    document.body.classList.toggle('asleep', asleep);
}

pause.addEventListener('click', async () =>
{
    const kept = await api.storage.local.get('asleep');
    const asleep = kept.asleep !== true;

    await api.storage.local.set({ asleep });
    wording(asleep);

    state.textContent = '';

    if (asleep)
    {
        line(said('paused'), 'stopped');
    }
    else
    {
        ask();
    }
});

opening();
