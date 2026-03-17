import "htmx.org";

// Repeatable fields: add/remove rows for list-type form fields.
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

// Contributor editing: person suggest, name split, kind toggle, link/unlink.
//
// Data attributes:
//   data-person-suggest     - contributor row container
//   data-suggest-input      - input that triggers person search (name, given, family)
//   data-suggest-url        - endpoint URL
//   data-suggest-results    - ul for results panel (to the right)
//   data-kind-toggle        - checkbox for person/organization
//   data-person-card        - read-only linked person display
//   data-unlink-person      - button to unlink person
//   data-contributor-fields - editable fields container (hidden when linked)
(function () {
	var debounceTimer;

	// Suggest on input — fires from name, given_name, or family_name fields.
	document.addEventListener("input", function (e) {
		var input = e.target.closest("[data-suggest-input]");
		if (!input) return;

		clearTimeout(debounceTimer);
		var container = input.closest("[data-person-suggest]");
		var results = container && container.querySelector("[data-suggest-results]");
		if (!results) return;

		// Don't search for organizations.
		var kindInput = container.querySelector('[name="contributors.kind"]');
		if (kindInput && kindInput.value === "organization") {
			results.innerHTML = "";
			return;
		}

		var query = input.value.trim();
		if (query.length < 2) {
			results.innerHTML = "";
			return;
		}

		debounceTimer = setTimeout(function () {
			var url = input.getAttribute("data-suggest-url") + "?q=" + encodeURIComponent(query);
			fetch(url)
				.then(function (r) {
					if (!r.ok) throw new Error("suggest: " + r.status);
					return r.json();
				})
				.then(function (people) {
					results.innerHTML = "";
					people.forEach(function (p) {
						var li = document.createElement("li");
						li.textContent = p.name;
						if (p.given_name || p.family_name) {
							var small = document.createElement("small");
							small.textContent = " (" + [p.given_name, p.family_name].filter(Boolean).join(" ") + ")";
							li.appendChild(small);
						}
						li.style.cursor = "pointer";
						li.addEventListener("click", function () {
							linkPerson(container, p);
							results.innerHTML = "";
						});
						results.appendChild(li);
					});
				})
				.catch(function (err) {
					console.error(err);
				});
		}, 300);
	});

	// Best-effort name split on blur of name field.
	document.addEventListener("focusout", function (e) {
		var input = e.target.closest("[data-suggest-input]");
		if (!input) return;

		var container = input.closest("[data-person-suggest]");
		if (!container) return;

		// Close suggest after short delay (allow click on result).
		var results = container.querySelector("[data-suggest-results]");
		if (results) {
			setTimeout(function () { results.innerHTML = ""; }, 200);
		}

		// Only split from the name field.
		if (input.getAttribute("name") !== "contributors.name") return;

		// Don't split org names.
		var kindInput = container.querySelector('[name="contributors.kind"]');
		if (kindInput && kindInput.value === "organization") return;

		var gn = container.querySelector('[name="contributors.given_name"]');
		var fn = container.querySelector('[name="contributors.family_name"]');
		if (!gn || !fn) return;

		// Only split if given/family are both empty.
		if (gn.value.trim() !== "" || fn.value.trim() !== "") return;

		var name = input.value.trim();
		if (!name) return;

		var parts = name.split(/\s+/);
		if (parts.length >= 2) {
			fn.value = parts.pop();
			gn.value = parts.join(" ");
		}
	});

	// Kind toggle: person ↔ organization.
	document.addEventListener("change", function (e) {
		var toggle = e.target.closest("[data-kind-toggle]");
		if (!toggle) return;

		var container = toggle.closest("[data-person-suggest]");
		if (!container) return;

		var isOrg = toggle.checked;

		var kindInput = container.querySelector('[name="contributors.kind"]');
		if (kindInput) {
			kindInput.value = isOrg ? "organization" : "person";
		}

		// Show/hide person-only fields.
		var personFields = container.querySelector("[data-person-fields]");
		if (personFields) {
			personFields.style.display = isOrg ? "none" : "";
		}

		if (isOrg) {
			// Clear person link, name parts, and results.
			unlinkPerson(container);
			var gn = container.querySelector('[name="contributors.given_name"]');
			var fn = container.querySelector('[name="contributors.family_name"]');
			if (gn) gn.value = "";
			if (fn) fn.value = "";
			var results = container.querySelector("[data-suggest-results]");
			if (results) results.innerHTML = "";
		}
	});

	// Unlink person button.
	document.addEventListener("click", function (e) {
		var btn = e.target.closest("[data-unlink-person]");
		if (!btn) return;

		var container = btn.closest("[data-person-suggest]");
		if (container) {
			unlinkPerson(container);
		}
	});

	function linkPerson(container, person) {
		var nameInput = container.querySelector('[name="contributors.name"]');
		var pid = container.querySelector('[name="contributors.person_id"]');
		var gn = container.querySelector('[name="contributors.given_name"]');
		var fn = container.querySelector('[name="contributors.family_name"]');

		if (nameInput) nameInput.value = person.name || "";
		if (pid) pid.value = person.id;
		if (gn) gn.value = person.given_name || "";
		if (fn) fn.value = person.family_name || "";

		// Show person card, hide fields.
		var card = container.querySelector("[data-person-card]");
		var fields = container.querySelector("[data-contributor-fields]");
		if (!card) {
			// Create card dynamically.
			card = document.createElement("div");
			card.setAttribute("data-person-card", "");
			var strong = document.createElement("strong");
			strong.textContent = person.name;
			card.appendChild(strong);
			if (person.given_name || person.family_name) {
				var span = document.createElement("span");
				span.textContent = " (" + [person.given_name, person.family_name].filter(Boolean).join(" ") + ")";
				card.appendChild(span);
			}
			card.appendChild(document.createTextNode(" "));
			var unlinkBtn = document.createElement("button");
			unlinkBtn.type = "button";
			unlinkBtn.setAttribute("data-unlink-person", "");
			unlinkBtn.textContent = container.getAttribute("data-unlink-label") || "Unlink";
			card.appendChild(unlinkBtn);
			if (fields) {
				fields.parentNode.insertBefore(card, fields);
			}
		}
		if (card) card.style.display = "";
		if (fields) fields.style.display = "none";
	}

	function unlinkPerson(container) {
		var pid = container.querySelector('[name="contributors.person_id"]');
		if (pid) pid.value = "";

		var card = container.querySelector("[data-person-card]");
		var fields = container.querySelector("[data-contributor-fields]");
		if (card) card.style.display = "none";
		if (fields) fields.style.display = "";
	}
})();
