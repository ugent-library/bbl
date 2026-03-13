#!/usr/bin/env python3
"""Convert legacy organization JSONL to ImportOrganizationInput JSONL format.

Usage:
    python3 scripts/convert-legacy-organizations.py < export_organizations.jsonl > organizations.jsonl
"""

import json
import re
import sys

# Root-level orgs (no parent_id) and their kinds.
ROOT_KINDS = {
    "UGent": "university",
    "UZGent": "hospital",
    "UGentAssoc": "association",
    "ResearchCenter": "research_center",
    "GUK": "campus",
}

# Direct children of UGent that are faculties (two-letter codes).
FACULTY_IDS = {
    "LW", "RE", "WE", "GE", "TW", "EB", "DI", "PP", "LA", "FW", "PS",
    "DS", "CA",
}

# Pattern: "(ceased 1-1-2010)" or "(ceased 01-10-2018)" or "(ceased)"
CEASED_RE = re.compile(r"\s*\(ceased(?:\s+(\d{1,2})-(\d{1,2})-(\d{4}))?\)\s*$")


def parse_ceased(name):
    """Extract end_date from name and return (clean_name, end_date_str or None)."""
    m = CEASED_RE.search(name)
    if not m:
        return name, None
    clean = name[:m.start()]
    if m.group(1):
        day, month, year = m.group(1), m.group(2), m.group(3)
        return clean, f"{year}-{month.zfill(2)}-{day.zfill(2)}T00:00:00Z"
    # "(ceased)" with no date
    return clean, None


def infer_kind(rec, parents):
    """Infer organization kind from position in the tree."""
    source_id = rec["id"]
    parent_id = rec.get("parent_id")

    # Explicit root kinds
    if source_id in ROOT_KINDS:
        return ROOT_KINDS[source_id]

    # No parent — unknown root
    if not parent_id:
        return "organization"

    # Direct child of UGent with known faculty code
    if parent_id == "UGent" and source_id in FACULTY_IDS:
        return "faculty"

    # University colleges under the association
    if parent_id == "UGentAssoc":
        return "university_college"

    # Children of a research center parent
    if parent_id == "ResearchCenter":
        return "research_center"

    # Everything else is a department
    return "department"


def convert(rec, parents):
    """Convert one legacy org record to ImportOrganizationInput."""
    source_id = rec["id"]
    name = rec.get("name", "")

    # Parse "(ceased ...)" from name
    clean_name, end_date = parse_ceased(name)

    out = {
        "source_id": source_id,
        "kind": infer_kind(rec, parents),
        "attrs": {},
        "identifiers": [],
        "rels": [],
    }

    if end_date:
        out["end_date"] = end_date

    if clean_name:
        out["attrs"]["names"] = [{"val": clean_name}]

    # Parent relationship
    parent_id = rec.get("parent_id")
    if parent_id:
        out["rels"].append({
            "ref": {"source_id": parent_id},
            "kind": "parent",
        })

    return out


def main():
    # Two-pass: first collect parent info, then convert.
    records = []
    parents = {}  # source_id -> parent_id
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        rec = json.loads(line)
        records.append(rec)
        parents[rec["id"]] = rec.get("parent_id")

    for rec in records:
        out = convert(rec, parents)
        print(json.dumps(out, ensure_ascii=False))


if __name__ == "__main__":
    main()
