Deflix Stremio addon
====================

[Deflix](https://deflix.tv) addon for [Stremio](https://stremio.com)

Automatically turns torrents into debrid/cached streams, for high speed and no seeding.

Currently supported providers:

- [x] <https://real-debrid.com>

> More providers will be supported in the future!

Run
---

The addon is a remote addon, so it's an HTTP web service. It's written in Go.

You can use one of the precompiled binaries from GitHub:

1. Download the binary for your OS from <https://github.com/doingodswork/deflix-stremio/releases>
2. Simply run the executable binary
3. To stop the program press `Ctrl-C` (or `⌘-C` on macOS)

Or use Docker:

1. `docker pull doingodswork/deflix-stremio`
2. `docker run --name deflix-stremio -p 8080:8080 doingodswork/deflix-stremio`
3. To stop the container: `docker stop deflix-stremio`

Use
---

After you started the web service with either the binary or Docker, it's running on `http://localhost:8080`.

Then:

1. Get your RealDebrid API token from <https://real-debrid.com/apitoken>
2. Enter the addon URL in the search box of the addons section of Stremio, like this:
   - `http://localhost:8080/YOUR_API_TOKEN/manifest.json`  
     > (replace `YOUR_API_TOKEN` by your actual API token!)

That's it!