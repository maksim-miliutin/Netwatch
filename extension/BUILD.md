# Building

Nothing is bundled or minified. `tsc` compiles each file in `src/` to a
matching file in `dist/`, one to one, so a reviewer can put the two side by
side and read them.

    npm install
    npm run build

For Firefox, rename `manifest.firefox.json` over `manifest.json`. It differs
in two things: the background runs as a script rather than a service worker,
and it carries an id of its own.

Node 20, TypeScript 5.