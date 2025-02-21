import htmx from "htmx.org/dist/htmx.esm.js";
import { initCallback as initBootstrap } from "bootstrap.native";
import initRepeatedFields from "./repeated_fields.js";
import initTags from "./tags.js";

htmx.config.defaultFocusScroll = true;
htmx.onLoad(initBootstrap);
htmx.onLoad(initRepeatedFields);
htmx.onLoad(initTags);
