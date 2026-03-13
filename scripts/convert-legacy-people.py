#!/usr/bin/env python3
"""Convert legacy person JSONL to ImportPersonInput JSONL format.

Usage:
    python3 scripts/convert-legacy-people.py < people.jsonl > people_bbl.jsonl
"""

import json
import sys


def convert(rec):
    """Convert one legacy person record to ImportPersonInput."""
    source_id = str(rec["_id"])

    first_name = rec.get("first_name", "")
    last_name = rec.get("last_name", "")

    # Build display name from parts; use full_name if available.
    name = rec.get("full_name", "")
    if not name and (first_name or last_name):
        name = f"{first_name} {last_name}".strip()

    out = {
        "source_id": source_id,
        "attrs": {},
        "identifiers": [],
    }

    if name:
        out["attrs"]["name"] = name
    if first_name:
        out["attrs"]["given_name"] = first_name
    if last_name:
        out["attrs"]["family_name"] = last_name

    # Identifiers
    for uid in rec.get("ugent_id", []):
        out["identifiers"].append({"scheme": "ugent_id", "val": str(uid)})

    orcid = rec.get("orcid_id")
    if orcid:
        out["identifiers"].append({"scheme": "orcid", "val": orcid})

    return out


def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        rec = json.loads(line)
        out = convert(rec)
        print(json.dumps(out, ensure_ascii=False))


if __name__ == "__main__":
    main()
