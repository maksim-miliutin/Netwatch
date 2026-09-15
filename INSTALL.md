# Installing

netwatch is two halves. The program keeps the list and talks to Discord; the
extension tells it what the browser is playing. Neither does anything alone.

## Three steps

**1. Unpack this archive** anywhere you like. `netwatch.exe` can sit wherever
it is convenient; it keeps its list beside itself, or in your home folder when
that is not writable.

**2. Load the extension.** In Chrome open `chrome://extensions`, turn on
Developer mode at the top right, press Load unpacked, and choose the
`extension` folder from this archive.

Chrome will show a bubble about developer mode extensions every time it
starts. That is Chrome being careful about anything it did not install itself,
not a sign of trouble.

**3. Run `netwatch.exe`.** A window opens with the list. Play something in the
browser and it turns up there within ten seconds.

If nothing turns up, reload the tab that is playing. Chrome does not put the
extension into pages that were already open.

## The Discord card

It works out of the box. What plays goes on your profile, and friends see it.
Discord itself has to be the desktop one, and open, with Settings, Activity,
"Display current activity as a status message" turned on.

To turn it off, or to use an application of your own, there is a box at the
foot of the page under Settings.

## Keeping it running

The page has a Start with Windows button. With it on, netwatch starts quietly
at login and sits by the clock. Left click the icon opens the window, right
click gives Open and Quit.

Closing the window does not stop anything. The program is not the window.

## Updating

There is no automatic update. Download the new release and unpack it over the
old one, then reload the extension in Chrome.

## Firefox

Install it from addons.mozilla.org instead of this archive, then run
`netwatch.exe` as above.

---

# Установка

netwatch состоит из двух половин. Программа ведёт список и говорит с Discord,
расширение сообщает ей, что играет в браузере. По отдельности они бесполезны.

## Три шага

**1. Распакуйте архив** куда угодно. `netwatch.exe` может лежать где удобно:
список он держит рядом с собой, а если туда нельзя писать, то в домашней папке.

**2. Загрузите расширение.** В Chrome откройте `chrome://extensions`, включите
режим разработчика справа сверху, нажмите «Загрузить распакованное» и выберите
папку `extension` из этого архива.

Chrome будет показывать пузырь про расширения в режиме разработчика при каждом
запуске. Так он относится ко всему, что установил не сам; это не поломка.

**3. Запустите `netwatch.exe`.** Откроется окно со списком. Включите что-нибудь
в браузере, и через десять секунд оно там появится.

Если не появилось, обновите вкладку, где играет. Chrome не заходит на
страницы, открытые до установки расширения.

## Карточка Discord

Работает сразу. То, что играет, показывается в вашем профиле, и это видят
друзья. Сам Discord должен быть настольным и открытым, а в его настройках, в
разделе «Активность», включён показ текущей активности.

Выключить или поставить своё приложение можно внизу страницы, в настройках.

## Чтобы работало всегда

На странице есть кнопка «Запускать с Windows». С ней netwatch стартует при
входе в систему тихо и садится к часам. Левый щелчок по значку открывает окно,
правый даёт «Открыть» и «Выход».

Закрытие окна ничего не останавливает: программа это не окно.

## Обновление

Автоматического нет. Скачайте новый выпуск, распакуйте поверх старого и
перезагрузите расширение в Chrome.

## Firefox

Ставится из addons.mozilla.org, а не из этого архива. Дальше так же:
запустить `netwatch.exe`.
