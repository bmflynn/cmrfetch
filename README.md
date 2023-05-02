# cmrfetch
[![Go](https://github.com/bmflynn/cmrfetch/actions/workflows/go.yml/badge.svg)](https://github.com/bmflynn/cmrfetch/actions/workflows/go.yml)
[![CodeQL](https://github.com/bmflynn/cmrfetch/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bmflynn/cmrfetch/actions/workflows/github-code-scanning/codeql)
[![Go Report Card](https://goreportcard.com/badge/github.com/bmflynn/cmrfetch)](https://goreportcard.com/report/github.com/bmflynn/cmrfetch)
[![Coverage Status](https://coveralls.io/repos/github/bmflynn/cmrfetch/badge.svg)](https://coveralls.io/github/bmflynn/cmrfetch)

CLI tool for searching NASA Earthdata CMR collection metadata and downloading granules.

The main purpose is to ease the discovery and download official NASA [EOSDIS](https://www.earthdata.nasa.gov/eosdis) products made available
by NASA EOSDIS [DAACs](https://www.earthdata.nasa.gov/eosdis/daacs) or [SIPS](https://www.earthdata.nasa.gov/eosdis/sips).

The NASA Earthdata CMR [Search API](https://cmr.earthdata.nasa.gov/search/site/docs/search/api.html)
has vast amounts of metadata available, and therefore the API results are very general. `cmrfetch` makes 
some assuptions regarding the metadata available in order to present it in a simplified manner. These 
assumptions may result in unexpected results for some collections or granules. If there are issues with 
metadata for a specific collection feel free to file an [issue](https://github.com/bmflynn/cmrfetch/issues).

## Search and Discovery
`cmrfetch` allows you to search for product collections using keywords, titles, shortnames, 
platform, instrument, etc.... To keep search sizes reasonable, at least one major search filter
must be specified. See `cmrfetch collections --help` for more information.

You can also search for collection granule metadata directly, however, because the set of
available granules is quite large you will get best results by being as specific with your
filtering as you can. 

## Downloading
`cmrfetch` will also download resulting granules. Currently, only granules hosted via HTTP
can be downloaded, but this may change to additionally support S3 ingest for DIRECT ACCESS
urls.

### Download Authentication
Most, if not all, data providers hosting granules require NASA Earthdata authentication,
([register](https://urs.earthdata.nasa.gov/users/new)). 

You may also need to [authorize](https://wiki.earthdata.nasa.gov/display/EL/How+To+Pre-authorize+an+application)
specific provider applications to be able to download from them. See the Earthdata Login [User Docs](https://urs.earthdata.nasa.gov/documentation/for_users)
for more information.

`cmrfetch` uses a netrc file (`~/.netrc` on linux/osx, `%USERPROFILE%/_netrc` on windows) for 
authentication. 

To setup netrc authentication, make sure you have the netrc file at the location specifed
above and add the following line, replacing `<LOGIN>` and `<PASSWORD>` with your account
information:
```
machine urs.earthdata.nasa.gov login <LOGIN> password <PASSWORD>
```
> **NOTE**: It is very important to make sure this file is not readable by other users.
  On linux/osx you can limit permissions like so `chmod 0600 ~/.netrc`

### Authentication Cookies
`cmrfetch` performs a single login for every instantiation. If you are downloading multiple
granules via a single instantiation it will store the authentication cookies in memory such 
that login is only performed once, but it does not save the cookies to disk.

This is contrary to `curl` or `wget` examples that you will see in the 
[documentation](https://urs.earthdata.nasa.gov/documentation/for_users).

## Keywords
There does not seem to be any canonical list of valid names for searchable resource names
such as `provier`, `platform`, and `instrument`. CMR does however support a simple keyword
search via their [Facet Auto-Complete](https://cmr.earthdata.nasa.gov/search/site/docs/search/api.html#autocomplete-facets)
service.

This is implemeted here via the `keywords` subcommand. My recommendtaion it to just start
searching for keywords you think may be relavent and hopefully you'll find what you are 
looking for.

## Output Formatting
Output is by default formatted to provide a basic level of data that is easily readable. Commands
generally provide additional output formats that will provide a greater level of detail or 
machine readable output such as JSON or CSV.

Note, however, that when search for granules the result set can be very large and the default
output format reads all results in to memory before formatting. So, when searching granules be
sure to provide as many filters as possible or perhaps choose an output format that handles 
streaming output, such as JSON or CSV.

## Error Handling
There is not a lot of direct error handling with regard to the format of input parameters. 
Instead, `cmrfetch` relies on the CMR API to return errors regarding input field/parameters.
If there are any issue that can be clarified with client-side error handling, please file
an issue.

## Examples

1. Search for Near Real-Time Cloud-Mask Collections
   ```
   $> cmrfetch -d nrt -s "cldmsk*"
   ┌────────────────────────────┬─────────┬───────────────────┬─────────────┬──────────┐
   │ SHORTNAME                  │ VERSION │ CONCEPT           │ REVISION_ID │ PROVIDER │
   ├────────────────────────────┼─────────┼───────────────────┼─────────────┼──────────┤
   │ CLDMSK_L2_VIIRS_NOAA20_NRT │ 1       │ C2003160566-ASIPS │ 3           │ ASIPS    │
   │ CLDMSK_L2_VIIRS_SNPP_NRT   │ 1       │ C1607563719-ASIPS │ 3           │ ASIPS    │
   └────────────────────────────┴─────────┴───────────────────┴─────────────┴──────────┘
   ```
2. Search for the NOAA-20 Level-2 Cloud Mask (`CLDMSK_L2_VIIRS_NOAA20`).
   
   First get the collection concept id using:
   ```
   $> cmrfetch collections -s CLDMSK_L2_VIIRS_NOAA20 
   ┌────────────────────────┬─────────┬───────────────────┬─────────────┬──────────┐
   │ SHORTNAME              │ VERSION │ CONCEPT           │ REVISION_ID │ PROVIDER │
   ├────────────────────────┼─────────┼───────────────────┼─────────────┼──────────┤
   │ CLDMSK_L2_VIIRS_NOAA20 │ 1       │ C1964798938-LAADS │ 6           │ LAADS    │
   └────────────────────────┴─────────┴───────────────────┴─────────────┴──────────┘
   ```
   Then use the collection concept id to get view some granules:
   ```
   $> cmrfetch granules -c C1964798938-LAADS -t 2023-04-01,2023-04-01T00:06:00Z
   ┌───────────────────────────────────────────────────────────┬─────────┬──────────────────┬───────────────────┬─────────────┐
   │ NAME                                                      │ SIZE    │ NATIVE_ID        │ CONCEPT_ID        │ REVISION_ID │
   ├───────────────────────────────────────────────────────────┼─────────┼──────────────────┼───────────────────┼─────────────┤
   │ CLDMSK_L2_VIIRS_NOAA20.A2023091.0006.001.2023091131339.nc │ 52.2 MB │ LAADS:7485481337 │ G2647214816-LAADS │ 1           │
   │ CLDMSK_L2_VIIRS_NOAA20.A2023091.0000.001.2023091131336.nc │ 43.3 MB │ LAADS:7485481318 │ G2647214807-LAADS │ 1           │
   │ CLDMSK_L2_VIIRS_NOAA20.A2023090.2354.001.2023091121321.nc │ 50.2 MB │ LAADS:7485462762 │ G2647210867-LAADS │ 1           │
   └───────────────────────────────────────────────────────────┴─────────┴──────────────────┴───────────────────┴─────────────┘
   ```
   Yep, looks good, let's go ahead and download those results to `./downloads`. By default 4 files 
   are downloaded concurrentlye (see `--download-concurrency`).
   ```
   $> cmrfetch granules -c C1964798938-LAADS -t 2023-04-01,2023-04-01T00:06:00Z --download ./downloads
   2023/05/01 13:49:35 fetched https://ladsweb.modaps.eosdis.nasa.gov/archive/allData/5110/CLDMSK_L2_VIIRS_NOAA20/2023/091/CLDMSK_L2_VIIRS_NOAA20.A2023091.0006.001.2023091131339.nc in 6.2s(64.1 Mb/s)
   2023/05/01 13:49:36 fetched https://ladsweb.modaps.eosdis.nasa.gov/archive/allData/5110/CLDMSK_L2_VIIRS_NOAA20/2023/090/CLDMSK_L2_VIIRS_NOAA20.A2023090.2354.001.2023091121321.nc in 6.5s(59.4 Mb/s)
   2023/05/01 13:49:46 fetched https://ladsweb.modaps.eosdis.nasa.gov/archive/allData/5110/CLDMSK_L2_VIIRS_NOAA20/2023/091/CLDMSK_L2_VIIRS_NOAA20.A2023091.0000.001.2023091131336.nc in 17.1s(19.3 Mb/s)
   ```
   Existing granules will be skipped.
   
3. View a specific granule, with more detail:
   ```
   $> cmrfetch granules -c C1964798938-LAADS -f CLDMSK_L2_VIIRS_NOAA20.A2023091.0006.001.2023091131339.nc -o long
   ┌─────────────────────┬───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
   │ name                │ CLDMSK_L2_VIIRS_NOAA20.A2023091.0006.001.2023091131339.nc                                                                                             │
   │ size                │ 52.2 MB                                                                                                                                               │
   │ checksum            │ fb792a683115ef192a9e33b0dd6b649c                                                                                                                      │
   │ checksum_alg        │ MD5                                                                                                                                                   │
   │ download_url        │ https://ladsweb.modaps.eosdis.nasa.gov/archive/allData/5110/CLDMSK_L2_VIIRS_NOAA20/2023/091/CLDMSK_L2_VIIRS_NOAA20.A2023091.0006.001.2023091131339.nc │
   │ native_id           │ LAADS:7485481337                                                                                                                                      │
   │ revision_id         │ 1                                                                                                                                                     │
   │ concept_id          │ G2647214816-LAADS                                                                                                                                     │
   │ collection          │ CLDMSK_L2_VIIRS_NOAA20/1                                                                                                                              │
   │ download_direct_url │ s3://prod-lads/CLDMSK_L2_VIIRS_NOAA20/CLDMSK_L2_VIIRS_NOAA20.A2023091.0006.001.2023091131339.nc                                                       │
   │ daynight            │ Day                                                                                                                                                   │
   │ timerange           │ [2023-04-01T00:06:00.000Z 2023-04-01T00:12:00.000Z]                                                                                                   │
   │ boundingbox         │ [-170.845001,-41.507393,-135.764923,-36.170902,-144.453003,-16.412685,-172.955856,-20.74267,-170.845001,-41.507393]                                   │
   └─────────────────────┴───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
   ```

References:

  * [NASA Eathdata](https://earthdata.nasa.gov)
  * [NASA Earthdata CMR Search API](https://cmr.earthdata.nasa.gov/search)
  * [NASA Earthdata Collection Directory](https://cmr.earthdata.nasa.gov/search/site/collections/directory/eosdis)
    - This is particularly useful as a reasonable browseable list of Providers and Collections.
  * NASA Global Change Master Directory -- Keywords
    - [Instruments](https://gcmd.earthdata.nasa.gov/KeywordViewer/scheme/instruments)
    - [Platforms](https://gcmd.earthdata.nasa.gov/KeywordViewer/scheme/platforms)

