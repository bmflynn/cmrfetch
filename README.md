# cmrfetch
An application to make it easier to search for and download granules discovered
using [NASA CMR Search API](https://cmr.earthdata.nasa.gov).

A [NASA Earthdata](https://earthdata.nasa.gov) [Account](https://urs.earthdata.nasa.gov)
is generally required to download data.

# Examples

# Lookup Collection ConceptID
Granules are searched using the collection concept ID, which can be looked up
using the `collections` sub-command. You will need to know the provider name,
e.g., ASIPS, LAADS, etc..., where the product resides.

The following will list all available collections available at the Atmosphere
SIPS:
```
cmrsearch collections --provider ASIPS
```

# Lookup Granule Metadata
A `granules` sub-command is also provided that will display simple granule metadata
without ingesting. It provides similar filter flags as the ingest command and
therefore can be useful to inspect files before ingesting or for debugging.

Note that credentials are not required for searching.
```
cmrfetch granules \
    --product=AERDB_L2_VIIRS_SNPP/1.1 \
    --temporal 2022-01-01T00:00:00Z,2022-01-02T00:00:00Z
```

Or alternatively, using concept id directly:
```
cmrfetch granules \
    --concept-id=C1560216482-LAADS \
    --temporal 2022-01-01T00:00:00Z,2022-01-02T00:00:00Z
```

# Unattended scheduled/forward-stream ingest
A typical use-case is ingesting all granules made available in forward-stream,
e.g., ingesting all Near-realtime data as it is made available.

This can be accomplished with `cmrfetch` using cron with dynamically generated
timerange over the last 72h.

Using a concept ID in a cronjob script something like:
```sh
#!/bin/bash
export EARTHDATA_USER=<username>
export EARTHDATA_PASSWD=<password>
export CONCEPT_ID=C1607549631-ASIPS

lastlog=/tmp/$(basename $0).log
lock=/tmp/$(basename $0).lock
statedir=$HOME/ingest
start=$(date -u -d "-72 hours" +%Y-%m-%dT%H:%M:%SZ)
end=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Use flock to prevent simultaneous instances; see flock manpage
(
    flock -n 9 || exit 1
    cmrsearch -c ${CONCEPT_ID} --temporal=${start},${end} --dir=${statedir} &> ${lastlog}

) 9>${lock}
```

All files will downloaded into `statedir` an any failed transfers will get a
`<name>.error` file containing some information regarding the error. Files are downloaded
to a temporary name until finished and moved into place. By default 2 files are downloaded
at a time.

Any files that already exist will not be downloaded.
