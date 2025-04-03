import htmx from "htmx.org/dist/htmx.esm.js";

const reTmpl = /^data-bb-tmpl-(.+)/;

export default function (rootEl) {
    rootEl
        .querySelectorAll("[data-bb-repeated-field-add]")
        .forEach((el) => el.addEventListener("click", addFormValue));
    rootEl
        .querySelectorAll("[data-bb-repeated-field-delete]")
        .forEach((el) => el.addEventListener("click", deleteFormValue));
}

function setValueIndex(formValue, valueIndex) {
    Array.from(formValue.getElementsByTagName("*")).forEach(function (el) {
        if (el.hasAttributes()) {
            let attrs = el.attributes;
            for (var i = 0; i < attrs.length; i++) {
                let m = attrs[i].name.match(reTmpl);
                if (m) {
                    el.setAttribute(m[1], attrs[i].value.replace("{i}", valueIndex));
                }
            }
        }
    });
}

// delete a value from the field
function deleteFormValue(e) {
    let formField = e.target.closest("[data-bb-repeated-fields]");
    e.target.closest("[data-bb-repeated-field]").remove();
    let length = Array.from(formField.children).length;

    for (var valueIndex = 0; valueIndex < length; valueIndex++) {
        setValueIndex(formField.children[valueIndex], valueIndex);
    }
}

// add a new value to the field
function addFormValue(e) {
    let formField = e.target.closest("[data-bb-repeated-fields]");
    let formValues = formField.querySelectorAll("[data-bb-repeated-field]");
    let lastValue = formValues[formValues.length - 1];
    let valueIndex = formValues.length;

    let newValue = lastValue.cloneNode(true);
    newValue.querySelectorAll(".form-control").forEach((item) => {
        item.value = "";
    });
    newValue.querySelectorAll(".is-invalid").forEach((item) => {
        item.classList.remove("is-invalid");
    });

    // set html attrs from their templates
    setValueIndex(newValue, valueIndex);

    // switch last value button to delete
    let lastBtn = lastValue.querySelector("[data-bb-repeated-field-add]");
    let classList = lastBtn.classList;
    lastBtn.removeAttribute("data-bb-repeated-field-add");
    classList.remove("btn-outline-primary");
    classList.add("btn-link-muted");
    lastBtn.setAttribute("data-bb-repeated-field-delete", "");
    classList = lastValue.querySelector("i.if-add").classList;
    classList.remove("if-add");
    classList.add("if-delete");
    lastValue.querySelector(
        "[data-bb-repeated-field-delete] .visually-hidden",
    ).textContent = "Delete";
    lastBtn.removeEventListener("click", addFormValue);
    lastBtn.addEventListener("click", deleteFormValue);

    // insert new value
    lastValue.after(newValue);
    // activate htmx on new element
    htmx.process(newValue);
    // activate add button
    newValue
        .querySelector("[data-bb-repeated-field-add]")
        .addEventListener("click", addFormValue);
}
