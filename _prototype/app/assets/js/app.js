import htmx from 'htmx.org';
import 'bootstrap';
import initCentrifuge from './centrifuge.js';
import initClipboard from './clipboard.js';
import initRepeatable from './repeatable.js';
import initTags from './tags.js';
import initUppy from './uppy.js';

// htmx.logAll();
htmx.config.defaultFocusScroll = true;
htmx.onLoad(initClipboard);
htmx.onLoad(initRepeatable);
htmx.onLoad(initTags);
htmx.onLoad(initUppy);

initCentrifuge(document.body);
