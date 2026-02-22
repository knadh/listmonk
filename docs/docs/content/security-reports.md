If you spot a security vulnerability in listmonk, please report it via GitHub [security advisories](__https://github.com/knadh/listmonk/security/advisories__).
### What not to report

The below listed scenarios are either not security vulnerabilities or are of acceptable risk. They keep getting reported unfortunately. Please refrain from doing so.

### SQL injection via subscriber query
The subscribers UI (and APIs) support issuing of arbitrary SQL expressions via a `query` parameter. While listmonk ensures that the queries are executed as readonly and has basic checks for target tables to prevent accidental side-effects, it is not really possible to prevent arbitrary Turing-complete SQL expressions from calling various Postgres functions. Postgres itself does not offer an easy way to allow/disallow specific functions.

That's why this feature is behind a special permission `subscribers:sql_query` and its risks are [clearly documented](__https://listmonk.app/docs/roles-and-permissions/#user-roles__). In a multi-user scenario, it is up to an admin to allow this permission to trusted users.

### Stored XSS via SVG
In addition to images, listmonk allows uploading of arbitrary file types, .html, .js, .svg, .* and does not transform or modify the files. That means, it is possible to have `<script>`s and other arbitrary content inside HTML and SVG files. It is not possible for listmonk to have special checks or transformations for various file types, and many environments legitimately want SVGs and other filetypes to be uploaded.

In a multi-user scenario, it is possible for an admin to decide what file types to allow (Admin -> Settings -> Media). In an environment where SVG (or any other type) is considered risky, they can simply be disallowed from being uploaded.

### Stored XSS in campaign HTML
listmonk is a full-fledged HTML content management system (similar to WordPress). It allows campaign messages to have arbitrary HTML, including `<script>`s, which is a legitimate use case in many environments. While campaign previews within the admin are iframe-sandboxed, when a campaign is published as a webpage (archive view), it will obviously execute whatever `<script>` it has. This is expected of a content management and publishing system. In a multi-user scenario, it is up to the admin to give appropriate permissions to trusted users if they deem this undesirable.

### ReDoS/DoS in templating
listmonk is a full-fledged HTML content management system (similar to WordPress) which provides the full power of the Go templating language in addition to bundling [Sprig functions](__https://masterminds.github.io/sprig/__). This allows executing Turing-complete code within templates where it is possible to write code that does loop-within-loop or memory-allocation code that runs within loops that could cause a DoS. There is no way to prevent this. In a multi-user scenario, it is up to the admin to give appropriate permissions to trusted users if they deem this undesirable.
