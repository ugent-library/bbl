import htmx from "htmx.org/dist/htmx.esm.js";
import "bootstrap";
import initTags from "./tags.js";

htmx.config.defaultFocusScroll = true;
htmx.onLoad(initTags);
