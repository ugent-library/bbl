import { Centrifuge } from 'centrifuge';
import htmx from 'htmx.org/dist/htmx.esm.js';

export default function (rootEl) {
    if (!rootEl.hasAttribute("data-centrifuge")) {
        return
    }

    const url = rootEl.getAttribute("data-centrifuge-url");
    const token = rootEl.getAttribute("data-centrifuge-token");
    const centrifuge = new Centrifuge(url, {
        token: token
    });

    centrifuge.on('publication', function (ctx) {
        htmx.swap("body", ctx.data.content, { swapStyle: 'none' });
    })

    centrifuge.connect();
}
