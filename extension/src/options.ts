// The program takes a -port flag, and until this existed the extension did not
// know that: it kept reporting to 7373 and went quiet with nothing to say.

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
