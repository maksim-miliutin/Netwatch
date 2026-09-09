const field = document.getElementById('port') as HTMLInputElement;
const button = document.getElementById('save') as HTMLButtonElement;
const told = document.getElementById('said') as HTMLElement;

chrome.storage.local.get('port').then((kept) =>
{
    field.value = String(kept.port ?? 7373);
});

button.addEventListener('click', async () =>
{
    const port = Number(field.value);

    if (!Number.isInteger(port) || port < 1 || port > 65535)
    {
        told.textContent = 'A port is a number between 1 and 65535.';

        return;
    }

    await chrome.storage.local.set({ port });

    told.textContent = 'Saved. netwatch is at 127.0.0.1:' + port + ' now.';
});

const sites = document.getElementById('sites') as HTMLElement;

function reach(host: string): string
{
    return 'https://' + host + '/*';
}

async function allowed(host: string): Promise<boolean>
{
    return chrome.permissions.contains({ origins: [reach(host)] });
}

async function allow(host: string): Promise<boolean>
{
    const given = await chrome.permissions.request({ origins: [reach(host)] });

    if (!given)
    {
        return false;
    }

    await chrome.scripting.registerContentScripts([
    {
        id: host,
        matches: [reach(host)],
        js: ['dist/page.js'],
        runAt: 'document_idle',
    }]);

    return true;
}

function row(host: string, name: string, given: boolean): HTMLElement
{
    const line = document.createElement('div');
    const what = document.createElement('b');
    const rest = document.createElement('span');

    line.className = 'site';
    what.textContent = name || host;
    rest.textContent = given ? 'allowed' : '';

    line.append(what);

    if (given)
    {
        line.append(rest);

        return line;
    }

    const ask = document.createElement('button');

    ask.textContent = 'Allow';
    ask.addEventListener('click', async () =>
    {
        if (await allow(host))
        {
            ask.replaceWith(rest);
            rest.textContent = 'allowed';
        }
    });

    line.append(ask);

    return line;
}

async function listing(): Promise<void>
{
    const kept = await chrome.storage.local.get('port');
    const port = kept.port ?? 7373;

    try
    {
        const answer = await fetch('http://127.0.0.1:' + port + '/api/mine');
        const own = await answer.json() as Record<string, string>;

        sites.textContent = '';

        const hosts = Object.keys(own);

        if (hosts.length === 0)
        {
            const none = document.createElement('p');

            none.textContent = 'None yet. Add one at the foot of the netwatch page.';
            sites.append(none);

            return;
        }

        for (const host of hosts.sort())
        {
            sites.append(row(host, own[host], await allowed(host)));
        }
    }
    catch
    {
        sites.textContent = 'netwatch is not running, so there is nothing to list.';
    }
}

listing();
