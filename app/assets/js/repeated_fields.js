import htmx from "htmx.org/dist/htmx.esm.js";

// NEW
const repeatedFieldAttr = 'data-bbl-repeated-field';
const repeatedFieldSelector = '[data-bbl-repeated-field]';
const removeSelector = '[data-bbl-remove]';
const addSelector = '[data-bbl-add]';
const clearValueSelector = '[data-bbl-clear-value]';

export default function (rootEl) {
    if (rootEl.matches(repeatedFieldSelector))
        initRepeatedField(rootEl);
    rootEl.querySelectorAll(repeatedFieldSelector).forEach((el) => initRepeatedField(el));
}

function initRepeatedField(fieldEl) {
    let fieldName = fieldEl.getAttribute(repeatedFieldAttr);

    // remove
    fieldEl.querySelectorAll(removeSelector).forEach((btnEl) => {
        btnEl.addEventListener('click', () => {
            let fields = document.querySelectorAll(`[${repeatedFieldAttr}='${fieldName}']`);
            // only clear if it's the last remaining field and it contains the add button
            if (fields.length == 1 && fieldEl.querySelectorAll(addSelector).length > 0) {
                clearField(fieldEl)
            } else {
                fieldEl.remove();
                setFieldIndices(fieldName);
            }
        })
    });

    // add
    fieldEl.querySelectorAll(addSelector).forEach((btnEl) => {
        btnEl.addEventListener('click', () => {
            let newField = fieldEl.cloneNode(true);
            clearField(newField);
            fieldEl.after(newField);
            setFieldIndices(fieldName);
            initRepeatedField(newField); // TODO why necessary? htmx.process should take care of this
            htmx.process(newField);
        })
    });
}

function clearField(fieldEl) {
    fieldEl.querySelectorAll(clearValueSelector).forEach((el) => {
        el.value = "";
    });
    // TODO make configurable
    fieldEl.querySelectorAll('.is-invalid').forEach((el) => {
        el.classList.remove('is-invalid');
    });
}

function setFieldIndices(fieldName) {
    document.querySelectorAll(`[${repeatedFieldAttr}='${fieldName}']`).forEach((el, idx) => {
        el.querySelectorAll(`[name^='${fieldName}[']`).forEach((formEl) => {
            const name = formEl.getAttribute('name');
            const beforeIdx = name.slice(0, fieldName.length + 1);
            const afterIdx = name.slice(name.indexOf(']'));
            const newName = beforeIdx + idx.toString() + afterIdx;
            formEl.setAttribute('name', newName);
        });
    });
}
