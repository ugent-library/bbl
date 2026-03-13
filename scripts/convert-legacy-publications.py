#!/usr/bin/env python3
"""Convert legacy publication JSONL to ImportWorkInput JSONL format.

Usage:
    python3 scripts/convert-legacy-publications.py < publications.jsonl > works.jsonl
"""

import json
import sys


def convert_status(old_status):
    """Map legacy status to bbl work status."""
    mapping = {
        "public": "public",
        "deleted": "deleted",
        "private": "private",
        "returned": "private",
    }
    return mapping.get(old_status, "private")


def convert_author(author):
    """Convert a legacy author entry to ImportWorkContributor."""
    out = {"roles": ["author"]}
    person_id = author.get("person_id")
    if person_id and is_ulid(person_id):
        out["person_ref"] = {"id": person_id}
    ep = author.get("external_person")
    if ep:
        out["name"] = ep.get("full_name", "")
        out["given_name"] = ep.get("first_name", "")
        out["family_name"] = ep.get("last_name", "")
    return out


def convert_supervisor(sup):
    """Convert a legacy supervisor entry to ImportWorkContributor."""
    out = {"roles": ["supervisor"]}
    person_id = sup.get("person_id")
    if person_id and is_ulid(person_id):
        out["person_ref"] = {"id": person_id}
    ep = sup.get("external_person")
    if ep:
        out["name"] = ep.get("full_name", "")
        out["given_name"] = ep.get("first_name", "")
        out["family_name"] = ep.get("last_name", "")
    return out


def convert_abstracts(abstracts):
    """Convert legacy abstract list to Text list."""
    if not abstracts:
        return []
    out = []
    for a in abstracts:
        text = a if isinstance(a, str) else a.get("text", "")
        lang = a.get("lang", "") if isinstance(a, dict) else ""
        if text:
            out.append({"lang": lang, "val": text})
    return out


def convert_lay_summaries(summaries):
    """Convert legacy lay_summary list to Text list."""
    if not summaries:
        return []
    out = []
    for s in summaries:
        text = s if isinstance(s, str) else s.get("text", "")
        lang = s.get("lang", "") if isinstance(s, dict) else ""
        if text:
            out.append({"lang": lang, "val": text})
    return out


def convert_titles(rec):
    """Build titles list from title + alternative_title."""
    titles = []
    main_title = rec.get("title", "")
    if main_title:
        titles.append({"lang": "", "val": main_title})
    for alt in rec.get("alternative_title", []) or []:
        if alt:
            titles.append({"lang": "", "val": alt})
    return titles


def convert_identifiers(rec):
    """Extract identifiers from various legacy fields."""
    idents = []
    doi = rec.get("doi")
    if doi:
        idents.append({"scheme": "doi", "val": doi})
    for issn in rec.get("issn", []) or []:
        idents.append({"scheme": "issn", "val": issn})
    for eissn in rec.get("eissn", []) or []:
        idents.append({"scheme": "eissn", "val": eissn})
    handle = rec.get("handle")
    if handle:
        idents.append({"scheme": "handle", "val": handle})
    wos_id = rec.get("wos_id")
    if wos_id:
        idents.append({"scheme": "wos", "val": wos_id})
    seen = set()
    out = []
    for i in idents:
        key = (i["scheme"], i["val"])
        if key not in seen:
            seen.add(key)
            out.append(i)
    return out


def convert_classifications(rec):
    """Extract classifications."""
    classifications = []
    cl = rec.get("classification")
    if cl:
        classifications.append({"scheme": "bbl", "val": cl})
    wos_type = rec.get("wos_type")
    if wos_type:
        classifications.append({"scheme": "wos_type", "val": wos_type})
    for rf in rec.get("research_field", []) or []:
        classifications.append({"scheme": "research_field", "val": rf})
    seen = set()
    out = []
    for c in classifications:
        key = (c["scheme"], c["val"])
        if key not in seen:
            seen.add(key)
            out.append(c)
    return out


def convert_pages(rec):
    """Build pages extent from page_first/page_last."""
    start = rec.get("page_first", "")
    end = rec.get("page_last", "")
    if start or end:
        return {"start": start, "end": end}
    return None


def is_ulid(s):
    """Check if a string is a valid ULID (26 chars, Crockford base32)."""
    if len(s) != 26:
        return False
    try:
        int(s, 32)
        return True
    except ValueError:
        return False


def convert(rec):
    """Convert one legacy record to ImportWorkInput."""
    legacy_id = rec["id"]
    out = {
        "source_id": legacy_id,
        "kind": rec.get("type", ""),
        "status": convert_status(rec.get("status", "")),
        "attrs": {},
        "identifiers": convert_identifiers(rec),
        "classifications": convert_classifications(rec),
        "contributors": [],
    }
    if is_ulid(legacy_id):
        out["id"] = legacy_id

    attrs = out["attrs"]

    # Titles
    titles = convert_titles(rec)
    if titles:
        attrs["titles"] = titles

    # Abstracts
    abstracts = convert_abstracts(rec.get("abstract"))
    if abstracts:
        attrs["abstracts"] = abstracts

    # Lay summaries
    lay_summaries = convert_lay_summaries(rec.get("lay_summary"))
    if lay_summaries:
        attrs["lay_summaries"] = lay_summaries

    # Keywords
    keywords = rec.get("keyword", [])
    if keywords:
        attrs["keywords"] = keywords

    # Journal fields
    journal = rec.get("publication", "")
    if journal:
        attrs["journal_title"] = journal
    journal_abbr = rec.get("publication_abbreviation", "")
    if journal_abbr:
        attrs["journal_abbreviation"] = journal_abbr

    # Volume / issue
    volume = rec.get("volume", "")
    if volume:
        attrs["volume"] = volume
    issue = rec.get("issue", "")
    if issue:
        attrs["issue"] = issue

    # Pages
    pages = convert_pages(rec)
    if pages:
        attrs["pages"] = pages

    # Total pages
    page_count = rec.get("page_count", "")
    if page_count:
        attrs["total_pages"] = page_count

    # Publication metadata
    pub_status = rec.get("publication_status", "")
    if pub_status:
        attrs["publication_status"] = pub_status
    year = rec.get("year", "")
    if year:
        attrs["publication_year"] = year
    place = rec.get("place_of_publication", "")
    if place:
        attrs["place_of_publication"] = place
    publisher = rec.get("publisher", "")
    if publisher:
        attrs["publisher"] = publisher

    # Contributors: authors
    for a in rec.get("author", []) or []:
        out["contributors"].append(convert_author(a))

    # Contributors: supervisors (dissertations)
    for s in rec.get("supervisor", []) or []:
        out["contributors"].append(convert_supervisor(s))

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
