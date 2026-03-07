const reIdx = /^[0-9]+/;

export default function (rootEl) {
    if (rootEl.matches('[data-repeatable]'))
        init(rootEl);
    rootEl.querySelectorAll('[data-repeatable]').forEach((el) => init(el));
}

function init(componentEl) {
    componentEl.querySelectorAll('[data-repeatable-field]').forEach((fieldEl) => {
        initField(componentEl, fieldEl);
    });
}

function initField(componentEl, fieldEl) {
    // remove
    fieldEl.querySelectorAll('[data-repeatable-remove]').forEach((btnEl) => {
        btnEl.addEventListener('click', () => {
            const numFields = componentEl.querySelectorAll('[data-repeatable-field]').length;
            // only clear if it's the last remaining field
            if (numFields == 1) {
                clearField(fieldEl)
            } else {
                fieldEl.remove();
                setFieldIndices(componentEl);
            }
        })
    });

    // add
    fieldEl.querySelectorAll('[data-repeatable-add]').forEach((btnEl) => {
        btnEl.addEventListener('click', () => {
            const newField = fieldEl.cloneNode(true);
            clearField(newField);
            fieldEl.after(newField);
            setFieldIndices(componentEl);
            initField(componentEl, newField);
        })
    });
}

function clearField(fieldEl) {
    fieldEl.querySelectorAll('[data-repeatable-clear]').forEach((el) => {
        el.value = "";
    });
    fieldEl.querySelectorAll('.is-invalid').forEach((el) => {
        el.classList.remove('is-invalid');
    });
}

function setFieldIndices(componentEl) {
    const namePrefix = componentEl.getAttribute('data-repeatable-name');
    const idPrefix = componentEl.getAttribute('data-repeatable-id');
    componentEl.querySelectorAll('[data-repeatable-field]').forEach((el, idx) => {
        el.querySelectorAll(`[name^='${namePrefix}[']`).forEach((formEl) => {
            const attr = formEl.getAttribute('name');
            const beforeIdx = attr.slice(0, namePrefix.length + 1);
            const afterIdx = attr.slice(namePrefix.length + 1).replace(reIdx, '');
            const newAttr = beforeIdx + idx.toString() + afterIdx;
            formEl.setAttribute('name', newAttr);
        });
        if (idPrefix) {
            el.querySelectorAll(`[id^='${idPrefix}-']`).forEach((formEl) => {
                const attr = formEl.getAttribute('id');
                const beforeIdx = attr.slice(0, idPrefix.length + 1);
                const afterIdx = attr.slice(idPrefix.length + 1);
                if (!reIdx.test(afterIdx)) {
                    return;
                }
                const newAttr = beforeIdx + idx.toString() + afterIdx.replace(reIdx, '');
                formEl.setAttribute('id', newAttr);
            });
            el.querySelectorAll(`[for^='${idPrefix}-']`).forEach((formEl) => {
                const attr = formEl.getAttribute('for');
                const beforeIdx = attr.slice(0, idPrefix.length + 1);
                const afterIdx = attr.slice(idPrefix.length + 1);
                if (!reIdx.test(afterIdx)) {
                    return;
                }
                const newAttr = beforeIdx + idx.toString() + afterIdx.replace(reIdx, '');
                formEl.setAttribute('for', newAttr);
            });
        }
    });
}
