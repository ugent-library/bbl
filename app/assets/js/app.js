import htmx from "htmx.org/dist/htmx.esm.js";
import "bootstrap";
import initRepeatedFields from "./repeated_fields.js";
import initTags from "./tags.js";

htmx.config.defaultFocusScroll = true;
htmx.onLoad(initRepeatedFields);
htmx.onLoad(initTags);
