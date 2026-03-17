import htmx from "htmx.org";

// Find a form field within a container by its suffix (e.g. "name" matches
// "contributors[0].name"). Works with both flat and indexed field names.
function fieldByName(container, suffix) {
	return container.querySelector('[name$=".' + suffix + '"]')
		|| container.querySelector('[name="' + suffix + '"]');
}

// --- Repeatable fields ---

function initRepeatable(rootEl) {
	rootEl.querySelectorAll("[data-repeatable]").forEach(function (el) {
		el.addEventListener("click", function (e) {
			var addBtn = e.target.closest("[data-add-item]");
			if (addBtn) {
				var template = el.querySelector("[data-item-template]");
				var items = el.querySelector("[data-repeatable-items]");
				if (template && items) {
					var clone = template.content.cloneNode(true);
					if (el.hasAttribute("data-index-names")) {
						var nextIdx = items.querySelectorAll("[data-repeatable-item]").length;
						clone.querySelectorAll("[name]").forEach(function (field) {
							field.name = field.name.replace("[-1]", "[" + nextIdx + "]");
						});
					}
					items.appendChild(clone);
					htmx.process(items.lastElementChild);
				}
				return;
			}

			var removeBtn = e.target.closest("[data-remove-item]");
			if (removeBtn) {
				var item = removeBtn.closest("[data-repeatable-item]");
				if (item) item.remove();
			}
		});
	});
}

// --- Person suggest ---

function initPersonSuggest(rootEl) {
	rootEl.querySelectorAll("[data-person-suggest]").forEach(function (el) {
		var debounceTimer;

		el.addEventListener("input", function (e) {
			var input = e.target.closest("[data-suggest-input]");
			if (!input) return;

			clearTimeout(debounceTimer);
			var results = el.querySelector("[data-suggest-results]");
			if (!results) return;

			var kindInput = fieldByName(el, "kind");
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
								linkPerson(el, p);
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

		el.addEventListener("focusout", function (e) {
			var input = e.target.closest("[data-suggest-input]");
			if (!input) return;

			var results = el.querySelector("[data-suggest-results]");
			if (results) {
				setTimeout(function () { results.innerHTML = ""; }, 200);
			}

			if (!input.name.endsWith(".name")) return;

			var kindInput = fieldByName(el, "kind");
			if (kindInput && kindInput.value === "organization") return;

			var gn = fieldByName(el, "given_name");
			var fn = fieldByName(el, "family_name");
			if (!gn || !fn) return;
			if (gn.value.trim() !== "" || fn.value.trim() !== "") return;

			var name = input.value.trim();
			if (!name) return;

			var parts = name.split(/\s+/);
			if (parts.length >= 2) {
				fn.value = parts.pop();
				gn.value = parts.join(" ");
			}
		});

		el.addEventListener("change", function (e) {
			var toggle = e.target.closest("[data-kind-toggle]");
			if (!toggle) return;

			var isOrg = toggle.checked;
			var kindInput = fieldByName(el, "kind");
			if (kindInput) kindInput.value = isOrg ? "organization" : "person";

			var personFields = el.querySelector("[data-person-fields]");
			if (personFields) personFields.style.display = isOrg ? "none" : "";

			if (isOrg) {
				unlinkPerson(el);
				var gn = fieldByName(el, "given_name");
				var fn = fieldByName(el, "family_name");
				if (gn) gn.value = "";
				if (fn) fn.value = "";
				var results = el.querySelector("[data-suggest-results]");
				if (results) results.innerHTML = "";
			}
		});

		el.addEventListener("click", function (e) {
			if (e.target.closest("[data-unlink-person]")) {
				unlinkPerson(el);
			}
		});

		function linkPerson(container, person) {
			var nameInput = fieldByName(container, "name");
			var pid = fieldByName(container, "person_id");
			var gn = fieldByName(container, "given_name");
			var fn = fieldByName(container, "family_name");

			if (nameInput) nameInput.value = person.name || "";
			if (pid) pid.value = person.id;
			if (gn) gn.value = person.given_name || "";
			if (fn) fn.value = person.family_name || "";

			var card = container.querySelector("[data-person-card]");
			var cardName = container.querySelector("[data-person-name]");
			var fields = container.querySelector("[data-contributor-fields]");
			if (cardName) cardName.textContent = person.name || "";
			if (card) card.style.display = "";
			if (fields) fields.style.display = "none";
		}

		function unlinkPerson(container) {
			var pid = fieldByName(container, "person_id");
			if (pid) pid.value = "";

			var card = container.querySelector("[data-person-card]");
			var fields = container.querySelector("[data-contributor-fields]");
			if (card) card.style.display = "none";
			if (fields) fields.style.display = "";
		}
	});
}

// --- Boot ---

htmx.onLoad(function (rootEl) {
	initRepeatable(rootEl);
	initPersonSuggest(rootEl);
});
