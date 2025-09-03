# bbl

## improvements compared to the old biblio ecosystem

* combines the whole biblio ecosystem in one small binary (authorities, backoffice, discovery, oai, api's, ...)
* split generic (bbl) and ugent specific parts (biblio)
* rich and consistent database model for all entities and their relations (works, work representations, files, projects, organizations, people, users)
* work types, fields used are configurable
* detailed change history for all entities
* handle duplicate people records robustly
* job engine for long running or recurring tasks (gathering candidate works, large exports, ...)
* remove limits on import, export, upload and download sizes
* direct to s3 uploads and downloads
* seamless index switching
* own query language
* avoid deep paging with cursors/search after
* push notifications to users and api's
* semantic search (TODO)
* use the ORCID v3 api (TODO)
* ORCID sync (TODO)

## to be decided

* resumable file uploads and side loading via tus.io

## support libraries developed

* bind
* catbird
* vo
* tonga
* oaipmh
* opensearchswitcher
* muxurl

## dependencies

* postgres
* opensearch
* s3 compatible store
* oidc provider
