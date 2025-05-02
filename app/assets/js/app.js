import htmx from 'htmx.org/dist/htmx.esm.js';
import 'bootstrap';
import initUppy from "./uppy.js";

htmx.config.defaultFocusScroll = true;
htmx.onLoad(initUppy);
