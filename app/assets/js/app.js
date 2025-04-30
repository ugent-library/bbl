import htmx from 'htmx.org/dist/htmx.esm.js';
import 'bootstrap';
import Uppy from '@uppy/core';
import DragDrop from '@uppy/drag-drop';

htmx.config.defaultFocusScroll = true;

htmx.onLoad(rootEl => {
    rootEl.querySelectorAll('[data-uppy-drag-drop]').forEach(el => {
        new Uppy().use(DragDrop, {
            target: el,
        });
    });
});
