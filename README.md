# cmrfetch
![Build & Test](https://github.com/bmflynn/cmrfetch/actions/workflows/go.yml/badge.svg)

CLI tool for searching NASA Earthdata CMR collection metadata and downloading granules.

The main purpose is to ease the discovery and download official NASA [EOSDIS](https://www.earthdata.nasa.gov/eosdis) products made available
by NASA EOSDIS [DAACs](https://www.earthdata.nasa.gov/eosdis/daacs) or [SIPS](https://www.earthdata.nasa.gov/eosdis/sips).

The NASA Earthdata CMR [Search API](https://cmr.earthdata.nasa.gov/search/site/docs/search/api.html)
has vast amounts of metadata available, and therefore the API results are very general. `cmrfetch` makes 
some assuptions regarding the metadata available in order to present it in a simplified manner. These 
assumption make result in unexpected results for some collections or granules. If there are issues with 
metadata for a specific collection feel free to file an [issue](https://github.com/bmflynn/cmrfetch/issues).

## Download Authentication
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
that login is only performed ones, but it does not save the cookies to disk.

This is contrary to `curl` or `wget` examples that you will see in the documentation.
