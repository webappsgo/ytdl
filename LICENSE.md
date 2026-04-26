MIT License

Copyright (c) 2026 casapps

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

---

## Third-Party Licenses

This software includes the following third-party libraries used by the
current Go build:

| Library | Version | License | Upstream license |
|---------|---------|---------|------------------|
| github.com/dustin/go-humanize | v1.0.1 | MIT | https://github.com/dustin/go-humanize/blob/v1.0.1/LICENSE |
| github.com/go-chi/chi/v5 | v5.2.0 | MIT | https://github.com/go-chi/chi/blob/v5.2.0/LICENSE |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause | https://github.com/google/uuid/blob/v1.6.0/LICENSE |
| github.com/gorilla/websocket | v1.5.3 | BSD-2-Clause | https://github.com/gorilla/websocket/blob/v1.5.3/LICENSE |
| github.com/remyoudompheng/bigfft | v0.0.0-20230129092748-24d4a6f8daec | BSD-3-Clause | https://github.com/remyoudompheng/bigfft/blob/24d4a6f8daec/LICENSE |
| github.com/robfig/cron/v3 | v3.0.1 | MIT | https://github.com/robfig/cron/blob/v3.0.1/LICENSE |
| golang.org/x/crypto | v0.31.0 | BSD-3-Clause | https://cs.opensource.google/go/x/crypto/+/v0.31.0:LICENSE |
| golang.org/x/sys | v0.28.0 | BSD-3-Clause | https://cs.opensource.google/go/x/sys/+/v0.28.0:LICENSE |
| golang.org/x/time/rate | v0.8.0 | BSD-3-Clause | https://cs.opensource.google/go/x/time/+/v0.8.0:LICENSE |
| gopkg.in/yaml.v3 | v3.0.1 | MIT | https://github.com/go-yaml/yaml/blob/v3.0.1/LICENSE |
| modernc.org/libc | v1.55.3 | BSD-3-Clause | https://gitlab.com/cznic/libc/blob/v1.55.3/LICENSE-GO |
| modernc.org/mathutil | v1.6.0 | BSD-3-Clause | https://gitlab.com/cznic/mathutil/-/blob/v1.6.0/LICENSE |
| modernc.org/memory | v1.8.0 | BSD-3-Clause | https://gitlab.com/cznic/memory/blob/v1.8.0/LICENSE-GO |
| modernc.org/sqlite | v1.34.5 | BSD-3-Clause | https://gitlab.com/cznic/sqlite/blob/v1.34.5/LICENSE |

License metadata was generated from the current module graph with
`go-licenses csv ./...`, with `modernc.org/mathutil` verified against its
upstream `LICENSE` file because its repository does not expose the license
under a name the scanner recognizes consistently.
