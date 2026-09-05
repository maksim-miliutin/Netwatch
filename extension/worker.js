// Tells netwatch what is open, and nothing anywhere else: the address below is
// the only one this may reach, and the manifest holds it to that.
const NETWATCH = 'http://127.0.0.1:7373/api/seen';

// A page is called "YouTube" for a moment and then gets the name of the video,
// so the list is only useful if the title is given time to settle.
const SETTLE_MS = 4000;

const waiting = new Map();

function tell(tab)
{
    if (!tab.url || !tab.url.startsWith('http'))
    {
        return;
    }

    clearTimeout(waiting.get(tab.id));

    waiting.set(tab.id, setTimeout(async () =>
    {
        waiting.delete(tab.id);

        try
        {
            await fetch(NETWATCH,
            {
                method: 'POST',
                headers: { 'content-type': 'application/json' },
                body: JSON.stringify({ url: tab.url, title: tab.title ?? '' }),
            });
        }
        catch
        {
            // netwatch is not running, and a browser should not complain about
            // a program somebody closed.
        }
    }, SETTLE_MS));
}

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
});
