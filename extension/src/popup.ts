// The only place that answers "is this thing working at all". The page cannot:
// somebody who opens it already knows where to look.

interface Now
{
    playing: boolean;
    service?: string;
    title?: string;
    by?: string;
    paused?: boolean;
    gone?: number;
    whole?: number;
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

        return;
    }

    line(on.title ?? '', 'what');
    line((on.by ? on.by + ' · ' : '') + on.service + (on.paused ? ' · paused' : ''), 'who');

    // A stream has nothing to draw a bar towards, and a paused one would draw
    // a bar that stands still.
    if (on.whole && !on.paused)
    {
        bar(on.gone ?? 0, on.whole);
    }
}

async function ask(): Promise<void>
{
    try
    {
        const answer = await fetch('http://127.0.0.1:7373/api/now');

        draw(await answer.json() as Now);
    }
    catch
    {
        line('netwatch is not running.', 'quiet');
    }
}

ask();
