#!/usr/bin/env python3
"""Convert legacy project JSONL to ImportProjectInput JSONL format.

Usage:
    python3 scripts/convert-legacy-projects.py < projects.jsonl > projects_bbl.jsonl
"""

import json
import sys


def convert(rec):
    """Convert one legacy project record to ImportProjectInput."""
    source_id = str(rec["_id"])

    out = {
        "source_id": source_id,
        "attrs": {},
        "identifiers": [],
    }

    # Title
    title = rec.get("title", "")
    if title:
        out["attrs"]["title"] = title

    # Description (abstract with HTML)
    abstract = rec.get("abstract", "")
    if abstract:
        out["attrs"]["description"] = abstract

    # Dates (ISO strings)
    start_date = rec.get("start_date")
    if start_date:
        out["start_date"] = str(start_date) + "T00:00:00Z"

    end_date = rec.get("end_date")
    if end_date:
        out["end_date"] = str(end_date) + "T00:00:00Z"

    # Identifiers
    gismo_id = rec.get("gismo_id")
    if gismo_id:
        out["identifiers"].append({"scheme": "gismo", "val": str(gismo_id)})

    iweto_id = rec.get("iweto_id")
    if iweto_id:
        out["identifiers"].append({"scheme": "iweto", "val": str(iweto_id)})

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
