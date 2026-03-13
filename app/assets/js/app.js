import "htmx.org";

// Repeatable fields: add/remove rows for list-type form fields.
// Uses data attributes on the DOM:
//   data-repeatable       - container fieldset
//   data-repeatable-items - div holding the current items
//   data-repeatable-item  - one item row
//   data-add-item         - "Add" button
//   data-remove-item      - "Remove" button on each row
//   data-item-template    - <template> with the empty row HTML
document.addEventListener("click", function (e) {
	const addBtn = e.target.closest("[data-add-item]");
	if (addBtn) {
		const container = addBtn.closest("[data-repeatable]");
		if (!container) return;
		const template = container.querySelector("[data-item-template]");
		const items = container.querySelector("[data-repeatable-items]");
		if (template && items) {
			const clone = template.content.cloneNode(true);
			items.appendChild(clone);
		}
		return;
	}

	const removeBtn = e.target.closest("[data-remove-item]");
	if (removeBtn) {
		const item = removeBtn.closest("[data-repeatable-item]");
		if (item) {
			item.remove();
		}
	}
});
