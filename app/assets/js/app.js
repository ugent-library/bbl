import htmx from 'htmx.org/dist/htmx.esm.js';
import 'bootstrap';
import initClipboard from "./clipboard.js";
import initTags from "./tags.js";
import initUppy from "./uppy.js";

htmx.logAll();
htmx.config.defaultFocusScroll = true;
htmx.onLoad(initClipboard);
htmx.onLoad(initTags);
htmx.onLoad(initUppy);
