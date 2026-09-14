// Firefox answers with promises through browser, Chrome through chrome. One
// name for both, loaded before anything that speaks to the browser.
const api: typeof chrome = (globalThis as { browser?: typeof chrome }).browser ?? chrome;
