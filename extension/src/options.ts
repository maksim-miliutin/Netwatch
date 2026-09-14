const field = document.getElementById('port') as HTMLInputElement;
const button = document.getElementById('save') as HTMLButtonElement;
const told = document.getElementById('said') as HTMLElement;

api.storage.local.get('port').then((kept) =>
{
    field.value = String(kept.port ?? 7373);
});

button.addEventListener('click', async () =>
{
    const port = Number(field.value);

    if (!Number.isInteger(port) || port < 1 || port > 65535)
    {
        told.textContent = said('a port');

        return;
    }

    await api.storage.local.set({ port });

    told.textContent = said('saved') + port + '.';
});

const sites = document.getElementById('sites') as HTMLElement;

function reach(host: string): string
{
    return 'https://' + host + '/*';
}

async function allowed(host: string): Promise<boolean>
{
    return api.permissions.contains({ origins: [reach(host)] });
}

async function allow(host: string): Promise<boolean>
{
    const given = await api.permissions.request({ origins: [reach(host)] });

    if (!given)
    {
        return false;
    }

    await api.scripting.registerContentScripts([
    {
        id: host,
        matches: [reach(host)],
        js: ['dist/api.js', 'dist/page.js'],
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
    rest.textContent = given ? said('allowed') : '';

    line.append(what);

    if (given)
    {
        line.append(rest);

        return line;
    }

    const ask = document.createElement('button');

    ask.textContent = said('allow');
    ask.addEventListener('click', async () =>
    {
        if (await allow(host))
        {
            ask.replaceWith(rest);
            rest.textContent = said('allowed');
        }
    });

    line.append(ask);

    return line;
}

async function listing(): Promise<void>
{
    const kept = await api.storage.local.get('port');
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

            none.textContent = said('none');
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
        sites.textContent = said('no netwatch');
    }
}

async function settingUp(): Promise<void>
{
    const kept = await api.storage.local.get('port');

    await learn(Number(kept.port ?? 7373));

    for (const [where, key] of [['port-said', 'port'], ['sites-said', 'sites'],
                                ['sites-what', 'sites.what'], ['save', 'save']])
    {
        const it = document.getElementById(where);

        if (it)
        {
            it.textContent = said(key);
        }
    }

    await listing();
}

settingUp();
