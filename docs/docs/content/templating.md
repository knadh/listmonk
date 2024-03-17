# Templating

A template is a re-usable HTML design that can be used across campaigns and transactional messages. Most commonly, templates have standard header and footer areas with logos and branding elements, where campaign content is inserted in the middle. listmonk supports Go template expressions that lets you create powerful, dynamic HTML templates.

listmonk supports [Go template](https://pkg.go.dev/text/template) expressions that lets you create powerful, dynamic HTML templates. It also integrates 100+ useful [Sprig template functions](https://masterminds.github.io/sprig/).

## Campaign templates
Campaign templates are used in an e-mail campaigns. These template are created and managed on the UI under `Campaigns -> Templates`, and are selected when creating new campaigns.

## Transactional templates
Transactional templates are used for sending arbitrary transactional messages using the transactional API. These template are created and managed on the UI under `Campaigns -> Templates`.

## Template expressions

There are several template functions and expressions that can be used in campaign and template bodies. They are written in the form `{{ .Subscriber.Email }}`, that is, an expression between double curly braces `{{` and `}}`.

### Subscriber fields

| Expression                    | Description                                                                                  |
| ----------------------------- | -------------------------------------------------------------------------------------------- |
| `{{ .Subscriber.UUID }}`      | The randomly generated unique ID of the subscriber                                           |
| `{{ .Subscriber.Email }}`     | E-mail ID of the subscriber                                                                  |
| `{{ .Subscriber.Name }}`      | Name of the subscriber                                                                       |
| `{{ .Subscriber.FirstName }}` | First name of the subscriber (automatically extracted from the name)                         |
| `{{ .Subscriber.LastName }}`  | Last name of the subscriber (automatically extracted from the name)                          |
| `{{ .Subscriber.Status }}`    | Status of the subscriber (enabled, disabled, blocklisted)                                    |
| `{{ .Subscriber.Attribs }}`   | Map of arbitrary attributes. Fields can be accessed with `.`, eg: `.Subscriber.Attribs.city` |
| `{{ .Subscriber.CreatedAt }}` | Timestamp when the subscriber was first added                                                |
| `{{ .Subscriber.UpdatedAt }}` | Timestamp when the subscriber was modified                                                   |

| Expression            | Description                                              |
| --------------------- | -------------------------------------------------------- |
| `{{ .Campaign.UUID }}`      | The randomly generated unique ID of the campaign         |
| `{{ .Campaign.Name }}`      | Internal name of the campaign                            |
| `{{ .Campaign.Subject }}`   | E-mail subject of the campaign                           |
| `{{ .Campaign.FromEmail }}` | The e-mail address from which the campaign is being sent |

### Functions

| Function                                    | Description                                                                                                                                                    |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `{{ Date "2006-01-01" }}`                   | Prints the current datetime for the given format expressed as a [Go date layout](https://yourbasic.org/golang/format-parse-string-time-date-example/) |
| `{{ TrackLink "https://link.com" }}` | Takes a URL and generates a tracking URL over it. For use in campaign bodies and templates.                                                                    |
| `https://link.com@TrackLink`         | Shorthand for `TrackLink`. Eg: `<a href="https://link.com@TrackLink">Link</a>`                                                                       |
| `{{ TrackView }}`                           | Inserts a single tracking pixel. Should only be used once, ideally in the template footer.                                                                     |
| `{{ UnsubscribeURL }}`                      | Unsubscription and Manage preferences URL. Ideal for use in the template footer.                                                                                                      |
| `{{ MessageURL }}`                          | URL to view the hosted version of an e-mail message.                                                                                                           |
| `{{ OptinURL }}`                            | URL to the double-optin confirmation page.                                                                                                                     |
| `{{ Safe "<!-- comment -->" }}`             | Add any HTML code as it is.                                                                                                                                   |

### Sprig functions
listmonk integrates the Sprig library that offers 100+ utility functions for working with strings, numbers, dates etc. that can be used in templating. Refer to the [Sprig documentation](https://masterminds.github.io/sprig/) for the full list of functions.


### Example template

The expression `{{ template "content" . }}` should appear exactly once in every template denoting the spot where an e-mail's content is inserted. Here's a sample HTML e-mail that has a fixed header and footer that inserts the content in the middle.

```html
<!DOCTYPE html>
<html>
  <head>
    <style>
      body {
        background: #eee;
        font-family: Arial, sans-serif;
        font-size: 6px;
        color: #111;
      }
      header {
        border-bottom: 1px solid #ddd;
        padding-bottom: 30px;
        margin-bottom: 30px;
      }
      .container {
        background: #fff;
        width: 450px;
        margin: 0 auto;
        padding: 30px;
      }
    </style>
  </head>
  <body>
    <section class="container">
      <header>
        <!-- This will appear in the header of all e-mails.
             The subscriber's name will be automatically inserted here. //-->
        Hi {{ .Subscriber.FirstName }}!
      </header>

      <!-- This is where the e-mail body will be inserted //-->
      <div class="content">
        {{ template "content" . }}
      </div>

      <footer>
        Copyright 2019. All rights Reserved.
      </footer>

      <!-- The tracking pixel will be inserted here //-->
      {{ TrackView }}
    </section>
  </body>
</html>
```

!!! info
    For use with plaintext campaigns, create a template with no HTML content and just the placeholder `{{ template "content" . }}`

### Example campaign body

Campaign bodies can be composed using the built-in WYSIWYG editor or as raw HTML documents. Assuming that the subscriber has a set of [attributes defined](querying-and-segmentation.md#sample-attributes), this example shows how to render those values in a campaign.

```
Hey, did you notice how the template showed your first name?
Your last name is {{.Subscriber.LastName }}.

You have done {{ .Subscriber.Attribs.projects }} projects.


{{ if eq .Subscriber.Attribs.city "Bengaluru" }}
  You live in Bangalore!
{{ else }}
  Where do you live?
{{ end }}


Here is a link for you to click that will be tracked.
<a href="{{ TrackLink "https://google.com" }}">Google</a>

```

The above example uses an `if` condition to show one of two messages depending on the value of a subscriber attribute. Many such dynamic expressions are possible with Go templating expressions.

## System templates
System templates are used for rendering public user-facing pages such as the subscription management page, and in automatically generated system e-mails such as the opt-in confirmation e-mail. These are bundled into listmonk but can be customized by copying the [static directory](https://github.com/knadh/listmonk/tree/master/static) locally, and passing its path to listmonk with the `./listmonk --static-dir=your/custom/path` flag.

You can fetch the static files with:<br>
`mkdir -p /home/ubuntu/listmonk/static ; wget -O - https://github.com/knadh/listmonk/archive/master.tar.gz | tar xz -C /home/ubuntu/listmonk/static --strip=2 "listmonk-master/static"`

[Docker example](https://yasoob.me/posts/setting-up-listmonk-opensource-newsletter-mailing/#custom-static-files), [binary example](https://github.com/knadh/listmonk/blob/master/listmonk-simple.service).


### Public pages

| /static/public/        |                                                          |
|------------------------|--------------------------------------------------------------------|
| `index.html`             | Base template with the header and footer that all pages use.        |
| `home.html`              | Landing page on the root domain with the login button.              |
| `message.html`           | Generic success / failure message page.                             |
| `optin.html`             | Opt-in confirmation page.                                           |
| `subscription.html`      | Subscription management page with options for data export and wipe. |
| `subscription-form.html` | List selection and subscription form page.                          |


To edit the appearance of the public pages using CSS and Javascript, head to Settings > Appearance > Public:

![image](https://user-images.githubusercontent.com/55474996/153739792-93074af6-d1dd-40aa-8cde-c02ea4bbb67b.png)



### System e-mails

| /static/email-templates/         |                                                                                                                                    |
|----------------------------------|------------------------------------------------------------------------------------------------------------------------------------|
| `base.html`                      | Base template with the header and footer that all system generated e-mails use.                                               |
| `campaign-status.html`           | E-mail notification that is sent to admins on campaign start, completion etc.                                                      |
| `import-status.html`             | E-mail notification that is sent to admins on finish of an import job.                                                             |
| `subscriber-data.html`           | E-mail that is sent to subscribers when they request a full dump of their private data.                                            |
| `subscriber-optin.html`          | Automatic opt-in confirmation e-mail that is sent to an unconfirmed subscriber when they are added.                                |
| `subscriber-optin-campaign.html` | E-mail content that's inserted into a campaign body when starting an opt-in campaign from the lists page.                          |
| `default.tpl`                    | Default campaign template that is created in Campaigns -> Templates when listmonk is first installed. This is not used after that. |

!!! info
    To turn system e-mail templates to plaintext, remove `<!doctype html>` from base.html and remove all HTML tags from the templates while retaining the Go templating code.
