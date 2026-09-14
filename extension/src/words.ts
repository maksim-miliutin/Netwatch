// The program decides the language and the popup asks it, so the two never
// drift apart. Kept here as well because a popup opened while nothing is
// running still has to say so.
const SAID: Record<string, Record<string, string>> =
{
    en:
    {
        'not running': 'netwatch is not running.',
        'nothing': 'Running. Nothing playing.',
        'paused': 'Paused. Nothing is being written down.',
        'pause': 'Pause recording',
        'start': 'Start recording',
        'open': 'Open the list',
        'port': 'The port netwatch listens on. Change it here if you started it with -port.',
        'save': 'Save',
        'saved': 'Saved. netwatch is at 127.0.0.1:',
        'a port': 'A port is a number between 1 and 65535.',
        'sites': 'Sites you added',
        'sites.what': 'netwatch knows them; the browser does not. It has to be asked for each one, and only you can ask it.',
        'allow': 'Allow',
        'allowed': 'allowed',
        'none': 'None yet. Add one at the foot of the netwatch page.',
        'no netwatch': 'netwatch is not running, so there is nothing to list.',
    },

    ru:
    {
        'not running': 'netwatch не запущен.',
        'nothing': 'Работает. Ничего не играет.',
        'paused': 'Пауза. Ничего не записывается.',
        'pause': 'Остановить запись',
        'start': 'Начать запись',
        'open': 'Открыть список',
        'port': 'Порт, который слушает netwatch. Поменяйте здесь, если запускали с -port.',
        'save': 'Сохранить',
        'saved': 'Сохранено. netwatch теперь на 127.0.0.1:',
        'a port': 'Порт — это число от 1 до 65535.',
        'sites': 'Добавленные сайты',
        'sites.what': 'netwatch о них знает, а браузер нет. Его надо просить о каждом, и просить можете только вы.',
        'allow': 'Разрешить',
        'allowed': 'разрешён',
        'none': 'Пока ни одного. Добавьте внизу страницы netwatch.',
        'no netwatch': 'netwatch не запущен, перечислять нечего.',
    },

    fr:
    {
        'not running': "netwatch n'est pas lancé.",
        'nothing': 'En marche. Rien ne joue.',
        'paused': 'En pause. Rien n\'est noté.',
        'pause': "Arrêter l'enregistrement",
        'start': "Reprendre l'enregistrement",
        'open': 'Ouvrir la liste',
        'port': 'Le port que netwatch écoute. Changez-le ici si vous l\'avez lancé avec -port.',
        'save': 'Enregistrer',
        'saved': 'Enregistré. netwatch est sur 127.0.0.1:',
        'a port': 'Un port est un nombre entre 1 et 65535.',
        'sites': 'Sites que vous avez ajoutés',
        'sites.what': "netwatch les connaît ; le navigateur non. Il faut le lui demander pour chacun, et vous seul pouvez le faire.",
        'allow': 'Autoriser',
        'allowed': 'autorisé',
        'none': 'Aucun pour l\'instant. Ajoutez-en un au bas de la page netwatch.',
        'no netwatch': "netwatch n'est pas lancé, il n'y a rien à lister.",
    },
};

let spoken = 'en';

function said(key: string): string
{
    return SAID[spoken]?.[key] ?? SAID.en[key] ?? key;
}

async function learn(port: number): Promise<void>
{
    try
    {
        const answer = await fetch('http://127.0.0.1:' + port + '/api/language');
        const told = await answer.json() as { in?: string };

        if (told.in && SAID[told.in])
        {
            spoken = told.in;
            await api.storage.local.set({ said: spoken });

            return;
        }
    }
    catch
    {
        // netwatch is not running, and the last language it named will do.
    }

    const kept = await api.storage.local.get('said');
    const last = String(kept.said ?? '');

    if (SAID[last])
    {
        spoken = last;
    }
}
