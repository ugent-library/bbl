import htmx from 'htmx.org/dist/htmx.esm.js';
import 'bootstrap';
import initUppy from "./uppy.js";

htmx.logAll();

htmx.config.defaultFocusScroll = true;

htmx.onLoad(initUppy);
