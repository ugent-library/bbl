import Tagify from '@yaireo/tagify';
import Sortable from 'sortablejs/modular/sortable.core.esm.js';

export default function (rootEl) {
    rootEl.querySelectorAll("[data-tags]").forEach((el) => {
        let inputName = el.dataset.tagsInputName

        let tagify = new Tagify(el, {
            delimiters: ",",
            duplicates: false,
            pasteAsTags: true,
        });

        tagify.on('change', function (evt) { 
            tagify.DOM.scope.parentElement.querySelectorAll(`input[name="${inputName}"]`).forEach((inputEl) => {
                inputEl.remove();
            });
            evt.detail.tagify.value.forEach((v) => {
                let inputEl = document.createElement("input");
                inputEl.type = "hidden";
                inputEl.name = inputName;
                inputEl.value = v.value;
                tagify.DOM.scope.parentElement.appendChild(inputEl);
            })
        });

        Sortable.create(tagify.DOM.scope, {
            draggable: '.' + tagify.settings.classNames.tag,
            onEnd: function () {
                tagify.updateValueByDOMTags()
            },
        });
    });
}
